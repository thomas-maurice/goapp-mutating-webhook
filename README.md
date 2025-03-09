# Go app mutating webhook

Inject `GOMAXPROCS` and `GOMEMLIMIT` dynamically into your Go containers based on the resources requirements from the deployment spec.

## Build

Compile the Docker image like so:
```bash
make docker
```

Then create a test k8s cluster with kind:
```bash
make kind
```

Once that's done install cert-manager:
```bash
make cert-manager
```

Once that's done install prometheus:
```bash
make prometheus
```

Then load the image into kind:
```bash
make load-image
```

And finally create the deployment:
```bash
make apply # might need to run this one a few times if the namespace originally doesn't exist
```

It will also create an example application in the default namespace. Change the spec to include `mutate: "true"` in the podspec template like so for example:
```yaml
kind: Deployment
apiVersion: apps/v1
metadata:
  name: sample-application
  namespace: default
spec:
  replicas: 3
  selector:
    matchLabels:
      app: sample-application
  template:
    # In the generated podspec you will also see some
    # new annotations, for example we'll add
    # an "adjusted-GOMEMLIMIT" that will be human
    # readable. i.e. "105 Mi" instead of "104857600"
    metadata:
      labels:
        app: sample-application
        # Make it "false" to deactivate the mutator
        mutate: "true"
    spec:
      containers:
        - name: sample-application
          image: alpine
          env:
            - name: GOMAXPROCS
              # In this example the value will be overwritten
              # to "1"
              value: "2"
            # This variable will be preserved (and so will
            # its order in the chain)
            - name: some_env
              value: foo
            # In the final podspec you will also see a new
            # GOMEMLIMIT env var
          command:
            - /bin/sh
          args:
            - -c
            - sleep 3600
          resources:
            requests:
              memory: 100Mi
              cpu: 100m
            limits:
              memory: 100Mi
              cpu: 100m
```

And re-apply, you should see your spawned pods showing up with `GOMAXPROCS` set according to the requests you set.

## Metrics

The mutator exposes metrics on the `metrics` port. One metric you might want to look at is
```
mutator_pod_mutations{namespace="default",status="STATUS"}
```

Where `STATUS` can be one of `SUCCESS` or `FAILURE`
