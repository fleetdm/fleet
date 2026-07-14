# gitops-auto-complete

Generates a JSON schema from Fleet's GitOps Go structs so
[yaml-language-server](https://github.com/redhat-developer/yaml-language-server) can
offer completion, hover docs, and validation while you write GitOps YAML.

## How to use

### Build the schema

Run from this directory (it reads Fleet's source via relative paths):

```bash
cd tools/gitops-auto-complete
go run . generated.schema.json   # write the schema (omit the arg to print to stdout)
```

Re-run it whenever the relevant Fleet structs change.

### Set up with yaml-language-server

Point yaml-language-server (used by Neovim, the VS Code YAML extension, and others)
at the generated file. Two ways:

- Map it to your GitOps files via the `yaml.schemas` setting — schema path → file globs.
- Or add a modeline to the top of a single file:

  ```yaml
  # yaml-language-server: $schema=/absolute/path/to/generated.schema.json
  ```

### Example: Neovim + lazy.nvim

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
            ["/absolute/path/to/generated.schema.json"] = {
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

Reflects a `GitOpsSpec` struct that mirrors the real top-level GitOps keys
(`org_settings`, `controls`, `software`, `policies`, ...), reusing Fleet's own types
for each section, via [`invopop/jsonschema`](https://github.com/invopop/jsonschema).
It then post-processes the result so the schema matches how GitOps files are actually
written — file-path references, legacy key aliases, required "source" keys, and field
docs pulled from Go comments.

It's kept as its own Go module (with a `replace` back to the repo) so it builds from
inside the repo without adding dependencies to the root `go.mod`.
