# MockGCP 

MockGCP contains stub implementations of GCP APIs, for use with testing KCC.  The implementations
don't do anything (there is no "backend") - creating a mock VM does not launch a VM.  Instead, the
goal is to provide just-enough consistent CRUD operations to test the KCC providers.
If the tests pass, we consider that to be a "good enough" mock for GCP.  When
we discover an issue that the mocks failed to catch, we should update mockgcp
to more accurately simulate GCP, ideally triggering the bug and then showing
that the fix addresses the issue.

At their core, most GCP APIs are CRUD-based, so implementing a basic implementation is fairly straightforward.
We are establishing some patterns to make the implementation as simple as possible, such as a simple
storage driver that works with the existing protos.

In general, we use the protos published by GCP at [https://github.com/googleapis/googleapis.git](https://github.com/googleapis/googleapis.git).  These protos make it relatively easy to implement the CRUD operations needed.

In addition, the protos have the annotations needed for GRPC <-> HTTP interop.
We use [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) to create HTTP servers from GRPC servers.

It is then fairly straightforward to inject our own HTTP handlers directly into the serving paths
for tests, so we don't even need to start a real webserver (i.e. we don't even need to listen on a port)

## Adding a new service

To add a new service, the easiest way is to copy one of the existing CLs. Example CL: [mockGCP for cloudfunctions](https://github.com/GoogleCloudPlatform/k8s-config-connector/pull/869)

Broadly the steps are:

1. Add the proto to the Makefile and run `make gen-proto` (or just `make`, as it's the default target).

   All google services are located in [googleapis Github repo](https://github.com/googleapis/googleapis/tree/master/google),
   refer to your resource's API documentation to identify the service name, for example [privateca](https://cloud.google.com/certificate-authority-service/docs/reference/rest#service:-privateca.googleapis.com).
   Once you identify the service, find the proper path to the proto files, for example:
   `cloud/security/privateca/v1/*.proto`. Then replace the prefix `googleapis/google/` to `./third_party/googleapis/mockgcp/`,
   and add into the Makefile.  If you're adding an API outside of googleapis/google/cloud,
   you may need to add commands to rename the API o mockgcp in [fixup-third-party.sh](fixup-third-party.sh]
1. Add a subdirectory called `mock<servicename>`.

   Copying one of the existing ones. `mockprivateca` is a reasonable basic one. Keep the files names.go and service.go,
   rename the third one to `<resourcename>.go`. 
1. Adjust your mock to use your protos (change the import paths), rename the types, etc.
1. Implement the core CRUD methods in `<resourcename>.go`.

   In the example file `capool.go`, we have GET, CREATE and DELETE implemented. More probably, you will need to implement
   some more CRUD methods.  In the log, you should see some errors indicating which method is not supported.
   For those methods, go into the service definition and implement a basic implementation - we have
   examples of most of the CRUD operations at this point.
1. Add your service to mock_http_roundtrip.go, something like `services = append(services, mockcloudbilling.New(env, storage))`.
1. Add the kind(s) you are adding support for to `func MaybeSkip` in harness.go.
1. Run the tests.

   Example command: `E2E_KUBE_TARGET=envtest RUN_E2E=1 E2E_GCP_TARGET=mock go test -test.count=1 -timeout 3600s -v ./tests/e2e -run TestAllInSeries/fixtures  2>&1 | tee log`.

   You can also specify a single test to run to save time, for example, `TestAllInSeries/fixtures/privatecacapool`, or
   reduce the timeout from 3600s to 600s or any other reasonable value depends on your test.

If you are lucky, everything will "just work" and you will see that your new tests are being invoked.

If something is not behaving as you would expect, you should be able to launch a debugger because it all runs in one process.
You can also use `ARTIFACTS=artifacts` to get detailed HTTP logs of the traffic, which is useful if you want to see the json requests & responses.
If you also use `E2E_GCP_TARGET=real` you can run against the real (non-mocked) GCP, and easily see what the actual behaviour should be.
Usually however, this is not necessary; the most common failure mode is that terraform or KCC expects a field to be automatically populated,
and it normally logs an error like "foo not set" (in this case, simply add that to your mock implementation.)