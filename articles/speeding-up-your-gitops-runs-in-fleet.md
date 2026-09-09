# Speeding up your GitOps runs in Fleet

As your Fleet deployment grows, GitOps run times can start to creep up. Longer runs mean slower feedback loops, delayed deployments, and frustrated engineers waiting for configuration changes to land. The good news is there are several practical strategies you can use to dramatically reduce your GitOps run times. This guide covers three key optimizations: organizing configurations with paths, using Fleet-maintained apps, and leveraging ETag-based conditional downloads.

## Use `include` with paths

When you manage a large number of policies, queries, or software packages, defining everything inline in your fleet YAML file becomes unwieldy and slow. Instead, use the `include` directive to reference configurations via file paths.

With path-based references, Fleet can process configurations more efficiently. Rather than parsing one massive YAML file, GitOps resolves each path independently, making it easier to manage at scale and faster to process.

### Example

Instead of defining all your patch policies inline:

```yaml
policies:
  - name: "macOS - Adobe Acrobat Reader"
    query: "..."
    # ... dozens more inline definitions
```

Reference them via paths:

```yaml
policies:
  - path: ./lib/macos-patch-policies.yml
  - path: ./lib/windows-patch-policies.yml
```

This approach also has organizational benefits. It makes your GitOps repository easier to navigate and allows teams to own specific configuration files without conflicts.

> You can see how Fleet uses path-based references for patch policies in the [Fleet GitOps repository](https://github.com/fleetdm/fleet/blob/main).

## Use Fleet-maintained apps instead of custom packages

Fleet-maintained apps (FMAs) are a significant performance improvement over custom packages for software deployment. The difference in GitOps run time can be dramatic.

### Why FMAs are faster

With **custom packages**, Fleet previously downloaded each package twice per run—once during the dry run and once during the actual run—regardless of whether the package had changed. For deployments managing dozens of software packages, this added up to substantial download time on every single GitOps run.

With **Fleet-maintained apps**, packages are only downloaded when the app has been updated. If nothing has changed, no download occurs, and the run completes much faster.

### How to switch

If you're currently using custom packages for software that Fleet maintains (common apps like Firefox, Chrome, Slack, Zoom, etc.), consider switching to the FMA equivalent. Check the [Fleet-maintained apps list](https://fleetdm.com/docs/using-fleet/fleet-maintained-apps) to see what's available.

### Current limitation

FMAs must be defined inline in the fleet file. You cannot reference them via a file path using `include`. This means the path-based organization strategy described above doesn't apply to FMAs yet. Keep this in mind when structuring your GitOps configuration.

## ETag-based conditional downloads for custom packages

For cases where you still need custom packages (internal tools, proprietary software, etc.), Fleet now supports conditional downloads using ETag headers. This means GitOps will skip re-downloading and re-uploading packages that haven't changed since the last run.

### How it works

1. When Fleet downloads a custom package for the first time, it stores the ETag header returned by the server.
2. On subsequent runs, Fleet sends a conditional request with the stored ETag.
3. If the server responds with `304 Not Modified`, the download is skipped entirely.
4. If the package has changed (new ETag), it gets downloaded as usual.

This optimization applies to both the dry run and the actual run, eliminating redundant downloads entirely.

### Requirements

ETag-based conditional downloads depend on the server hosting your package supporting ETag headers. Most modern hosting solutions support ETags out of the box:

- **Amazon S3**: Supports ETags by default
- **Google Cloud Storage**: Supports ETags by default
- **Azure Blob Storage**: Supports ETags by default
- **GitHub Releases**: Supports ETags
- **Your own web server (Nginx, Apache)**: Typically supports ETags for static files by default

### Tips

- **If you control the hosting** (e.g., your own S3 bucket), you can ensure ETag support is enabled and working correctly.
- **If you use a third-party URL**, verify that the server returns ETag headers. You can test this with a simple curl command:

```bash
curl -I https://example.com/path/to/package.pkg | grep -i etag
```

If you see an `ETag` header in the response, conditional downloads will work for that URL.

- **If your server doesn't support ETags**, the package will be downloaded on every run (the old behavior). Consider switching to a hosting solution that supports ETags or using Fleet-maintained apps where available.

## Summary

| Strategy | Impact | When to use |
|----------|--------|-------------|
| Path-based `include` | Better organization, faster parsing | Large configurations with many policies/queries |
| Fleet-maintained apps | Eliminates unnecessary downloads | Common software available in the FMA catalog |
| ETag conditional downloads | Skips unchanged custom packages | Custom/proprietary packages you host yourself |

By combining these strategies, you can significantly reduce your GitOps run times—especially at scale where dozens of packages and hundreds of policies are managed through Fleet.

<meta name="articleTitle" value="Speeding up your GitOps runs in Fleet">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="authorFullName" value="Mike McNeil">
<meta name="publishedOn" value="2026-05-30">
<meta name="category" value="guides">
<meta name="description" value="Practical strategies to reduce Fleet GitOps run times using path-based includes, Fleet-maintained apps, and ETag conditional downloads.">
