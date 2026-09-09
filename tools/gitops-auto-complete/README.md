# gitops-auto-complete

Generates a JSON schema from Fleet's GitOps Go structs so
[yaml-language-server](https://github.com/redhat-developer/yaml-language-server) can
offer completion, hover docs, and validation while you write GitOps YAML.

## How to use

### Build the schema

The tool is a separate Go module, so run it from its own directory:

```bash
cd tools/gitops-auto-complete
go run . generated-schema.json
```

The argument is the output file, or omit it to print to stdout. Re-run it whenever
the relevant Fleet structs change.

### Set up with yaml-language-server

Point yaml-language-server at the generated file. It's used by Neovim, the VS Code
YAML extension, and others. There are two ways to do this:

- Map it to your GitOps files with the `yaml.schemas` setting, which maps a schema
  path to file globs.
- Or add a modeline to the top of a single file:

  ```yaml
  # yaml-language-server: $schema=/absolute/path/to/generated-schema.json
  ```

### Neovim and lazy.nvim example

```lua
{
  "neovim/nvim-lspconfig",
  dependencies = {
    { "mason-org/mason.nvim", opts = {} },
    "mason-org/mason-lspconfig.nvim",
  },
  config = function()
    vim.lsp.config("yamlls", {
      settings = {
        yaml = {
          schemas = {
            -- schema file -> which YAML files it applies to
            ["/absolute/path/to/generated-schema.json"] = {
              "**/default.yml",
              "**/teams/*.yml",
              "**/fleets/*.yml",
            },
          },
        },
      },
    })
    vim.lsp.enable("yamlls")
  end,
}
```

Install the server once (`:MasonInstall yaml-language-server`), reload, and open a
GitOps file. Hover a key with `K` to see its type and docs.

## How it works

Reflects a `GitOpsSpec` struct that mirrors the real top-level GitOps keys, such as
`org_settings`, `controls`, `software`, and `policies`, reusing Fleet's own types for
each section, via [`invopop/jsonschema`](https://github.com/invopop/jsonschema). It
then post-processes the result so the schema matches how GitOps files are written:
file-path references, legacy key aliases, required fields for an item, and field docs
pulled from Go comments.

It's a separate Go module with a `replace` back to the repo, so it builds from inside
the repo without adding dependencies to the root `go.mod`.

## Known limitations

- The schema is filename-agnostic, but Fleet applies some keys differently by file.
  For example, `agent_options` and `reports` are rejected in `no-team.yml` and the
  unassigned file. The schema still accepts them there, so that mistake shows up at
  `fleetctl` apply time, not in the editor.
- `GitOpsSpec` and `ControlsWithTypes` are hand-written mirrors of `spec.GitOps` and
  `spec.GitOpsControls`, because those spec structs are untyped or untagged and reflect
  poorly. `TestControlsKeysCoverSpec` catches a controls-key drift, but a new top-level
  key has to be added to `GitOpsSpec` by hand, as `custom_host_vitals` was.
