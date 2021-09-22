### Fleet docs

#### Adding a link to Fleet docs
You can link documentation pages to each other using relative paths. For example, in `docs/1-Using-Fleet/1-Fleet-UI.md`, you can link to `docs/1-Using-Fleet/9-Permissions.md` by writing `[permissions](./9-Permissions.md)`. This will be automatically transformed into the appropriate URL for `fleetdm.com/docs`.

However, the `fleetdm.com/docs` compilation process does not account for relative links to directories **outside** of `/docs`.
Therefore, when adding a link to Fleet docs, it is important to always use the absolute file path.

#### Linking to a location on GitHub
When adding a link to a location on GitHub that is outside of `/docs`, be sure to use the canonical form of the URL.

To do this, navigate to the file's location on GitHub, and press "y" to transform the URL into its canonical form.

#### How to fix a broken link
For instances in which a broken link is discovered on fleetdm.com, check if the link is a relative link to a directory outside of `/docs`. 

An example of a link that lives outside of `/docs` is:

```
../../tools/app/prometheus
```

If the link lives outside `/docs`, head to the file's location on GitHub (in this case, [https://github.com/fleetdm/fleet/blob/main/tools/app/prometheus.yml)](https://github.com/fleetdm/fleet/blob/main/tools/app/prometheus.yml)), and press "y" to transform the URL into its canonical form ([https://github.com/fleetdm/fleet/blob/194ad5963b0d55bdf976aa93f3de6cabd590c97a/tools/app/prometheus.yml](https://github.com/fleetdm/fleet/blob/194ad5963b0d55bdf976aa93f3de6cabd590c97a/tools/app/prometheus.yml)). Replace the relative link with this link in the markdown file.

> Note that the instructions above also apply to adding links in the Fleet handbook.