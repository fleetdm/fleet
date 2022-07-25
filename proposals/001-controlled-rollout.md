# Controlled Rollout proposal

## Why

New features are great, everybody loves them. However, new features come by the hand of new code. New code can have bugs 
or it can have performance regressions.

Features aren't perfect for all users. Sometimes they are perfect for some users and not others. Sometimes they are 
perfect for some hosts and not others.

Rolling out a feature shouldn't always be a binary choice: enabled/disabled. In an ideal world, all features would be 
enabled by default, everybody would love them all, and they would work flawlessly for all possible use cases.

We are in the real world, though, which is not ideal. So we should give people running Fleet tools to rollout features 
slowly, so that they can update infrastructure if needed, or only use a feature within the scope that is useful for 
them.

This is a proposal on how this tool could look like and work.

## How

We would create a new type of boolean value in our `AppConfig` called `RolloutBoolean`.

`RolloutBoolean` will have a function `Get(h *fleet.Host) bool`. So instead of doing this:

```go 
if ac.HostSettings.EnableHostUsers {
    ...
}
```

We would do:

```go 
if ac.HostSettings.EnableHostUsers.Get(host) {
    ...
}
```

In yaml terms, this is what `RolloutBoolean` would be able to parse:

1. Regular true/false, 0/1, yes/no values

```yaml
---
apiVersion: v1
kind: config
spec:
  host_settings:
    enable_software_inventory: false
```

2. Only enable a feature for certain teams:

```yaml
---
apiVersion: v1
kind: config
spec:
  host_settings:
    enable_software_inventory:
      default: false
      overrides:
        teams:
        - team1: true
        - team2: true
```

2. Enabled for all except a specified team:

```yaml
---
apiVersion: v1
kind: config
spec:
  host_settings:
    enable_software_inventory:
      default: true
      overrides:
        teams:
        - team1: false
```

3. Enabled only for specific hosts:

```yaml
---
apiVersion: v1
kind: config
spec:
  host_settings:
    enable_software_inventory:
      default: false
      overrides:
        host_ids:
        - 3214: true
```

4. Enabled for hosts on a specific platform (as reported by osquery, not in terms of label membership):

```yaml
---
apiVersion: v1
kind: config
spec:
  host_settings:
    enable_software_inventory:
      default: false
      overrides:
        platforms:
        - linux: true
```

The `Get(h *Host) bool` function will use the provided host to define whether the feature is enabled or not based on
how it's defined in the configuration.