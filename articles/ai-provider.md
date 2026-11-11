# Use your own AI provider

By default, Fleet's [AI profile builder](https://fleetdm.com/guides/custom-os-settings) uses Fleet's built-in AI. If your organization requires an approved AI provider, you can connect your own Anthropic or OpenAI account instead.

When a provider is configured, Fleet only uses that provider. If the provider is unavailable, profile generation fails with an error. Fleet doesn't fall back to its built-in AI.

## Get an API key

Fleet connects to your provider with an API key. Both Anthropic and OpenAI show the full key only once, when it's created. If you didn't save an existing key, create a new one.

We recommend creating a dedicated key for Fleet, so you can track its usage and revoke it without affecting other tools.

### Anthropic

1. Sign in to the [Claude Console](https://platform.claude.com/).
2. Go to the [API keys](https://platform.claude.com/settings/keys) page.
3. Select **Create key** and name the key (for example, "Fleet").
4. Set **Linked account** to a service account and select a workspace to scope the key to.
5. Copy the key, which starts with `sk-ant-`, and store it somewhere safe, like a secrets manager.

Use a service account key instead of a personal key. Personal keys stop working if the person who created them leaves your organization.

If **Create key** is disabled, your role might not allow you to create keys. Ask an organization admin in the Claude Console to create a service account key for you.

### OpenAI

1. Sign in to the [OpenAI Platform](https://platform.openai.com/).
2. Select the project you want Fleet to use from the project selector. To keep Fleet's usage separate, create a new project.
3. Go to organization settings, select the project, and open **Members**.
4. Select **+ Service account**, enter an ID (for example, "fleet"), and select **Create**.
5. Copy the secret key and store it somewhere safe, like a secrets manager.

Creating service accounts requires the organization owner or project owner role. If you don't see the option, ask an owner to create the key for you.

If your account doesn't have billing set up, OpenAI might not let you create or use a key. Check **Billing** in the OpenAI Platform.

## Connect your provider

1. In Fleet, go to **Settings** > **Organization settings** > **AI and MCP**.
2. Select your provider: **Anthropic** or **OpenAI**.
3. Enter your API key.
4. Enter a model ID for each complexity level. Use the model ID from your provider's documentation, like `claude-opus-5-5`, not the model's display name.
5. Select **Save**.

Fleet checks your Anthropic model IDs when you save. If a model isn't found, the field is shown in red with an error.

After you save, the API key is hidden. To change it, enter a new key.

## Complexity

Fleet uses a different model depending on how complex the request is:

- **Low**: description, name, and Android profile generation.
- **High**: pre-profile generation checks and Windows and macOS profile generation.

Fleet sends the matching effort level to your provider with each request. Models that don't support effort are used without it. These models support effort:

- **Anthropic**: Claude Fable 5.x, Claude Opus 4.5 and later, Claude Sonnet 4.6 and later, and Claude Haiku 5.5.
- **OpenAI**: Reasoning models, including the GPT-5 family and o-series models.

## GitOps

To configure your provider in GitOps, use the [`ai_provider`](https://fleetdm.com/docs/configuration/yaml-files#ai-provider) section in your `default.yml`. Store your API key as an environment variable, not in plain text.

```yaml
org_settings:
  ai_provider:
    provider: anthropic
    api_key: $AI_PROVIDER_API_KEY
    models:
      low: claude-haiku-5-5
      high: claude-opus-5-5
```

## Troubleshooting

If profile generation fails, Fleet shows an error explaining why:

- **API key isn't valid**: The key was revoked, expired, or entered incorrectly. Create a new key and update it in Fleet.
- **No available credits**: Add credits or update billing in your provider's console.
- **Model wasn't found**: The provider may have retired the model. Update the model ID in Fleet.
- **Couldn't connect** or **rate limiting requests**: The provider is temporarily unavailable. Try again later.

Only global admins can change AI provider settings. If you're a maintainer, contact your Fleet administrator.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="TODO">
<meta name="authorFullName" value="TODO">
<meta name="publishedOn" value="TODO">
<meta name="articleTitle" value="Use your own AI provider">
<meta name="description" value="Learn how to connect Anthropic or OpenAI to Fleet's AI profile builder.">
