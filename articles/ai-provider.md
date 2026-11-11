# Use a custom AI provider

By default, Fleet's [AI profile builder](https://fleetdm.com/guides/custom-os-settings) uses Fleet's built-in AI. If your organization requires an approved AI provider, you can connect a custom AI provider, Anthropic or OpenAI, instead.

When a custom AI provider is configured, Fleet only uses that provider. If the provider is unavailable, profile generation fails with an error. Fleet doesn't fall back to its built-in AI.

## Get an API key

Fleet connects to your provider with an API key, the same way it connects to other third-party services. Both Anthropic and OpenAI show the full key only once, when it's created. If you didn't save an existing key, create a new one.

We recommend creating a dedicated key for Fleet, so you can track its usage and revoke it without affecting other tools.

### Anthropic

1. Sign in to the [Claude Console](https://platform.claude.com/).

2. Go to the [API keys](https://platform.claude.com/settings/keys) page.

3. Select **Create key** and name the key (for example, "Fleet").

4. Set **Expiration** to **Never**, so profile generation doesn't stop working when the key expires.

5. Set **Linked account** to a service account.

6. Scope the key to **Default workspace**.

7. Copy the key, which starts with `sk-ant-`, and store it somewhere safe, like a secrets manager.

Use a service account key instead of a personal key. Personal keys stop working if the person who created them leaves your organization.

If **Create key** is disabled, your role might not allow you to create keys. Ask an organization admin in the Claude Console to create a service account key for you.

### OpenAI

1. Sign in to the [OpenAI Platform](https://platform.openai.com/).

2. Go to the **API keys** page.

3. Select **Create new secret key**.

4. Set **Owned by** to **Service account**.

5. Enter a **Service Key Name** (for example, "Fleet").

6. Set **Project** to **Default project**.

7. Set **Expiration** to **Never**, so profile generation doesn't stop working when the key expires.

8. Select **Create secret key**.

9. Copy the key and store it somewhere safe, like a secrets manager.

Use a service account key instead of a user key. User keys are tied to a person and must be revoked when they leave your organization.

Service account keys have full access to their project. To limit what the key can access, create a separate project for Fleet and select it in step 6.

Creating service account keys requires the organization owner or project owner role. If you don't see the option, ask an owner to create the key for you.

If your account doesn't have billing set up, OpenAI might not let you create or use a key. Check **Billing** in the OpenAI Platform.

## Connect a custom AI provider

1. In Fleet, go to **Settings** > **Organization settings** > **AI and MCP**.

2. Select your provider: **Anthropic** or **OpenAI**.

3. Enter your API key.

4. Enter a model ID for each complexity level, like claude-opus-5-5. Use the model ID, not the model’s display name. Find current model IDs on the [Anthropic models page](https://platform.claude.com/docs/en/models/overview) or the [OpenAI models page](https://developers.openai.com/api/docs/models/all).

5. Select **Save**.

> OpenAI’s models page also lists image, video, and audio models, and models meant for ChatGPT. These won’t work with Fleet. Use a text model, such as a GPT-5 model.

Fleet checks your model IDs when you save. If a model isn't found, the field is shown in red with an error.

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

Only global admins can change custom AI provider settings. If you're a maintainer, contact your Fleet administrator.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="melpike">
<meta name="authorFullName" value="Mel Pike">
<meta name="publishedOn" value="2026-11-11">
<meta name="articleTitle" value="Use your own AI provider">
<meta name="description" value="Learn how to connect a custom AI provider, Anthropic or OpenAI, to Fleet's AI profile builder.">
