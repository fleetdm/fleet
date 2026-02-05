# Update Instructions for goval-dictionary

These are instructions for pulling in the latest changes from the upstream version of this library.
The `UPSTREAM_COMMIT` file tracks the upstream version that we last synced with.

_Notes:_

- Update `/path/to/fleet` below to your fleet repo location

## Fleet-specific changes

No changes have been made yet to the upstream goval-dictionary code.

## Update process

```bash
export FLEET_REPO=/path/to/fleet

# Clone upstream
git clone https://github.com/vulsio/goval-dictionary.git ~/goval-dictionary-merge
cd ~/goval-dictionary-merge

# Check out the last upstream commit we vendored
git checkout $(cat "$FLEET_REPO"/third_party/goval-dictionary/UPSTREAM_COMMIT)

# Create a branch for our downstream changes
git checkout -b fleet-changes

# Copy current vendored version into this working repo
rsync -a --delete "$FLEET_REPO"/third_party/goval-dictionary/ ./ --exclude .git
git add .
git commit -m "Apply Fleet downstream changes"

# Fetch upstream updates and merge them
git fetch origin
git checkout master
git merge origin/master
git checkout fleet-changes
git merge master   # resolve conflicts if any

# Copy merged result back into monorepo
rsync -a --delete ./ "$FLEET_REPO"/third_party/goval-dictionary/ --exclude .git

# Record the new upstream commit
git rev-parse origin/master > "$FLEET_REPO"/third_party/goval-dictionary/UPSTREAM_COMMIT

# Restore this file (rsync --delete removes it)
# Copy UPDATE_INSTRUCTIONS back or use: git checkout HEAD~1 -- UPDATE_INSTRUCTIONS

# Clean up
cd ~
rm -rf ~/goval-dictionary-merge
```

## Testing

After updating, verify the changes work correctly:

```bash
cd "$FLEET_REPO"/third_party/goval-dictionary
make test
```

## Removal

Once the arch support changes are merged upstream into vulsio/goval-dictionary,
this vendored copy can be removed and Fleet can depend on the upstream version directly.
