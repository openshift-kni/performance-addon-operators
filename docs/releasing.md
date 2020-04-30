# Releasing

> Note: this is about upstream releases only

Releases of the performance addon operator and its registry image are triggered by pushing git tags.
Since CSV version is using semantic versions, tag names need to follow semantic versioning as well.

Performance addon operator releases follow the `major.minor.patch` version of OKD / Openshift releases.
That means that its releases have the same `major.minor.patch` version, but it will have an additonal `-x`,
which will be incremented for each release, e.g. `4.5.0-1`

> Note: the CSV version number does only allow the `major.minor.patch` pattern. While images will be tagged with
> the complete version, the CSV version will be stripped to `major.minor.patch`.

## Release artifacts

The primary release artifact of the performance addon operator is the git tree.
The source code and selected build artifacts are available for download at:
https://github.com/openshift-kni/performance-addon-operators/releases

Pre-built containers are published on quay.io and can be viewed at:
https://quay.io/organization/openshift-kni

## Implementation details

Releases are done by drone.io CI. See its configuration at [.drone.yml](../drone.yaml).
The build process as identical as possible to Openshift CI. It does run some basic image validation steps,
but no e2e tests. But releases are always going to be made only on commits which passed Openshift CI.

### CI preparation

The release process needs some secretes which have to be prepared at [drone.io](https://cloud.drone.io)

- QUAY_USER: the user for pushing images, needs write access to all affected image repositories
- QUAY_PASSWORD: the password of the quay user
- GITHUB_TOKEN: a github OAuth token with public_repo access for creating the releases

### Triggering a release

#### Test builds

If you push a git tag with `test` in its name, it will build images, but NOT create the github release.
This might be useful for debugging build issues. Note that you still need to follow semver pattern, so
use e.g. `4.5.0-x-test1` (with `x` being the next build number).

#### Pre-releases

If needed, you can create pre releases by pushing tags like `4.5.0-x.rc1`. This will build and push
images with the given tag and create a github pre-release.

#### Releases

Releases are triggered by pushing tags like `4.5.0-x`. This will build and push
images with the given tag AND with a floating `4.5.0` tag, and create a github pre-release. You need
to complete the github release note and make it a release manually.
