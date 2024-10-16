# Fleet codebase index

## Directories

### articles/
Contains articles published on https://fleetdm.com. Examples include release notes (published for
each Fleet release) and guides (quick descriptions of Fleet features, written by engineers).

### assets/
Contains assets (images, fonts, styling) that are built into the Fleet UI. 

### build/
This directory is gitignored, but it contains build artifacts like the Fleet and `fleetctl` binaries.

### changes/
Contains [changes files](./docs/Contributing/Committing-Changes.md#changes-files). These files are
compiled into the release notes when we release a new version of Fleet.

### charts/
Contains Helm charts for running Fleet and a TUF server.

### cmd/
Contains the code for various command-line programs, including the [Fleet server](./cmd/fleet/) and [`fleetctl`](./cmd/fleetctl/). 

### docs/
Fleet documentation. The docs on https://fleetdm.com/docs are generated from
[docs/Get started](./docs/Get%20started/), [docs/Deploy](./docs/Deploy/),
[docs/Configuration](./docs/Configuration/), and [docs/REST API](./docs/REST%20API/)

The [docs/Contributing](./docs/Contributing/) directory contains docs for contributors, both interal
and external. If you're new on the Fleet engineering team, this is a great place to get started!

### ee/
Contains code for Fleet Premium. Any features that are Fleet Premium only should go in this directory.

### frontend/
Contains the code for the Fleet UI web app. Check out the [README](./frontend/README.md) for more
information on working with this code!

### git-hooks/
Contains some helpful [git hooks](https://git-scm.com/book/ms/v2/Customizing-Git-Git-Hooks) for
folks that are working on Fleet.

### handbook/
Contains the Fleet handbook, accessible at https://fleetdm.com/handbook. The handbook contains
Fleet's processes and describes how Fleet the business operates.

### infrastructure/

### it-and-security/

### node_modules/

### orbit/

### pkg/

### proposals/

### schema/

### scripts/

### server/

### terraform/

### test/

### test_tuf/

### tmp/

### tools/

### website/

