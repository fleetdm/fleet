# Application deployment

## Plan

User story:
"Allow admin users to define a set of apps that will be automatically installed on hosts (on a per-team basis)."

## Munki

Fleet will use Munki for application deployment to macOS devices.
See [here](https://github.com/munki/munki/wiki/Overview) for a basic overview of what Munki does and does not do.

Server side, Munki stores its components as files on a "repository":
- `pkgs/`: Actual installer items (`*.pkg`s, `*.dmg`s, etc.).
- `pkgsinfo/`: plists that describes installer items.
- `icons/`: icon png files.
- `catalogs/`: Each catalog is a set of apps (built from `pkgsinfo/`).
- `manifests/`: defines what each client should install (contains one or more `catalog` and a list of apps to install, remove, etc.).

Fleet could integrate with Munki in two ways:
A. Fleet manages the Munki repository 100%, generates repository files like `catalogs` & `manifests` on-the-fly.
B. Munki repository 100% managed by munki tools. Fleet just serves the files.
C. Hybrid (Fleet manages some parts of the repository but somehow allows power users to use Munki features directly)

I'll assume we'll just do (A) because of simplicity and UX:
- If we allow multiple ways to run Fleet with Munki then we will have to build different UIs depending on whether the admin set (A), (B) or (C).
- (A) allows for better UX as the user does not need to know Munki concepts to use Fleet.

## MVP-Dogfood deliverable

- Application management via `fleetctl` commands.
- No teams support. Configured apps will be installed globally ("global team").
- Munki offers other features like "managed_updates", "managed_uninstalls", etc., but we'll provide two ways of deployment at first:
  - "managed" (forced) installs
  - "optional" (self service) installs
- Hosted and not-Hosted installers
  - Hosted: Installers will be stored in an S3 bucket (thus an S3 object store will be a dependency).
  - TBD: We could leave not-hosted installers for a later iteration (not on MVP-Dogfood).
- No Fleet DM hosted catalog of installers (all installers must be manually uploaded to Fleet or an external service, aka non-hosted).

## Munki deployment

In the current PoC, after MDM enrollment of a device, Fleet will auto-push the following commands:
- `"InstallApplication"` to install a signed vanilla Munki.
- `"InstallProfile"` to:
  - Configure Munki to connect to Fleet for software deployment.
  - Set `ClientIdentifier` (to a random UUID) in the Munki profile, to allow fleet to identify the Munki client.
- We will use HTTP Basic Authentication + the assigned random `ClientIdentifier` to authenticate Munki clients to Fleet.

Once Munki is up and running, then it can be used to install practically any kind of application.

## Backend design

New MySQL table `apple_installers` that represents hosted-installers:
- `id`
- `name VARCHAR(255)` (UNIQUE): Name of the installer item (used by Munki protocol).
- `sha256 VARCHAR(64)` (UNIQUE): SHA-256 of the installer item.
- `pkginfo TEXT`: plist with default values automatically generated from the package contents.
- `hosted BOOLEAN`

If `hosted=true` then an entry in `apple_installers` will map to the following S3 storage paths (prefix `apple-mdm/apps/`):
- `apple-mdm/apps/pkgs/<id>`
- `apple-mdm/apps/icons/<id>`
    
> NOTE: We will have to generate `icons/_icon_hashes.plist` on-the-fly (used by Munki client)...
> With a combination of caching and the `ListObjectsV2` API.

New table `apple_installments` that represents allocation of installers into teams:
- `id`
- `installer_id` (reference to `apple_installers.id`)
- `managed BOOLEAN` (if `true`, then the app is automatically installed, if `false` then it will show up as optional in "Managed Software Center")
- `team` (`0` for Global)

## Fleetctl commands

Example of creating a hosted installer:
```sh
# User uploads installer item
fleetctl apple-mdm installer upload --installer=some-app.pkg
Installer uploaded with id=1, hash=c3e9e2ec300d231ee6e9cdfe1dd8fc03d62ac6c23d8ddfcc9358e430dca73ea4.

# User fetches the uploaded item
fleetctl apple-mdm installer get --id=1
<Output installer `apple_installers` entry, including `pkginfo` to stdout>

#
# User stores output of `pkginfo` on a `some-app.plist` and edits fields.
#
vim some-app.plist

# User overrides default pkginfo with custom `some-app.plist`
fleetctl apple-mdm installer set-pkginfo --id=1 --pkginfo=some-app.plist

# User adds installer item to a Team (or Globally)
fleetctl apple-mdm installments add --id=1 --optional={true|false} --team=Foo
```

Example of creating a not-hosted installer (using `PackageCompleteURL`):
```sh
# User generates pkginfo file: osquery-5.5.1.plist 
/usr/local/munki/makepkginfo ~/Downloads/osquery-5.5.1.pkg > osquery-5.5.1.plist

# User edits `osquery-5.5.1.plist` modifies `PackageCompleteURL`
# to point to e.g. https://github.com/osquery/osquery/releases/download/5.5.1/osquery-5.5.1.pkg
vim osquery-5.5.1.plist

# User imports application using pkginfo.
fleetctl apple-mdm installer import --pkginfo=osquery-5.5.1.plist
Installer imported with id=2, hash=c3e9e2ec300d231ee6e9cdfe1dd8fc03d62ac6c23d8ddfcc9358e430dca73ea4.

# User adds installer item to a Team (or Globally)
fleetctl apple-mdm installments add --id=2 --optional={true|false} --team=Foo
```

### Installers and Installments

- "Installer" is an installer item.
- "Installment" represents the addition of an installer item to a Team.

### Installers

`fleetctl apple-mdm installer upload --installer=*` (via API `POST */upload`):
1. Uploads pkg to storage.
2. Generates a pkginfo.
3. Stores entry in `apple_installers` with `hosted=true`.
4. Outputs <INSTALLER_ID>.

`fleetctl apple-mdm installer import --pkginfo=*`:
Stores entry in `apple_installers` with `hosted=false`.

`fleetctl apple-mdm installer get --id=<INSTALLER_ID>`:
Returns info of entry in `apple_installers`

`fleetctl apple-mdm installer list`
Lists entries in `apple_installers`.

`fleetctl apple-mdm installer set-pkginfo --id=<INSTALLER_ID> --pkginfo=some-app.plist`
Updates `pkginfo` of entry in `apple_installers`.

`fleetctl apple-mdm installer delete --id=<INSTALLER_ID>`
Remove entry from `apple_installers`

### Installments

`fleetctl apple-mdm installments add --id=<INSTALLER_ID> --team=<TEAM_ID>`:
1. Stores entry in `apple_installments`.
2. Returns <INSTALLMENT_ID>

`fleetctl apple-mdm installments get --id=<INSTALLMENT_ID>`
List entries in `apple_installments`

`fleetctl apple-mdm installments list --team=Foo`
List entries in `apple_installments`

`fleetctl apple-mdm installments delete --id=<INSTALLMENT_ID>`
Remove entry from `apple_installments`

## Catalogs and Manifests in Fleet

From https://github.com/munki/munki/wiki/Pkginfo-Files:

> It's important to remember that Munki clients never use pkginfo files directly -- they only query catalogs.
> Catalogs are constructed from pkginfo files. Therefore, any changes to pkginfo files (adding a new one, deleting one, or editing one)
> require rebuilding the catalogs using the makecatalogs tool.

- Fleet will generate only one `catalog`, called `"fleet"` on-the-fly from entries in `apple_installers`. (We will need to mimick the `makecatalogs` tool.)
- Fleet will generate `manifests` for each client on-the-fly from entries in `apple_installments` (Munki clients request for manifests using `<REPO_PATH>/manifests/<ClientIdentifier>`).
Fleet will determine what software needs to be installed (as managed or optional) on each client by looking at the host's team and entries in `apple_installments`. (We will need to mimick the `manifestutil` tool.)