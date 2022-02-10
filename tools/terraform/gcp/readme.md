### Pushing the Fleet image into Google Artifact registry

Login with gcloud helper:

```shell
gcloud auth configure-docker \
    us-central1-docker.pkg.dev
```

Pull latest image

`docker pull <latest fleet version>` for example `docker pull fleetdm/fleet:v4.9.1`

Tag it

```
docker tag fleetdm/fleet:v4.9.1 us-central1-docker.pkg.dev/<project_id>/fleet-repository/fleet:v4.9.1
```

Push to Google Artifact registry

`docker push us-central1-docker.pkg.dev/<project_id>/fleet-repository/fleet:v4.9.1`