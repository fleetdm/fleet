using System;
using System.Diagnostics;
using System.IO;
using System.Threading.Tasks;
using System.Windows;
using System.Windows.Input;
using System.Windows.Media;
using Microsoft.Web.WebView2.Core;

namespace FleetDesktop;

/// <summary>
/// WebView2-backed browser window. Port of BrowserWindow.swift.
///
/// Lifecycle:
///   var w = new MainWindow();
///   w.Preload(uri);   // starts loading without showing the window
///   w.Show();         // brings the window forward (loading overlay shown until page loads)
///
/// The WebView is kept alive between Show()/Hide() calls so reopening is instant.
/// </summary>
public partial class MainWindow : Window
{
    /// <summary>Window title shown in the title bar.</summary>
    public const string WindowTitle = "Fleet Desktop";

    /// <summary>URL schemes safe to open externally.</summary>
    private static readonly HashSet<string> AllowedExternalSchemes = new(StringComparer.OrdinalIgnoreCase)
    {
        "https", "http", "mailto",
    };

    /// <summary>Raised when a navigation error occurs (network or expired-token error page).</summary>
    public event Action? NavigationError;

    /// <summary>Raised the first time the window is shown (used to kick a token refresh).</summary>
    public event Action? WindowShown;

    private string? _fleetHost;
    private Uri? _homeUrl;
    private bool _coreReady;
    private int? _pendingStatusCode;
    private string? _pendingNavigationUri;

    /// <summary>JavaScript to run the next time a Fleet-host page finishes loading. Consumed once.
    /// Used by fleet://update_all and fleet://install_all to click in-page buttons.</summary>
    private string? _pendingPostLoadJs;

    /// <summary>
    /// Host of the external IdP page an SSO/auth flow is currently on. Non-null
    /// while a flow is in progress; external redirects are kept in the WebView so
    /// the full redirect chain completes in-app, but navigation is restricted to
    /// this host (hops to a new host are only allowed via server redirects or
    /// scripted navigations such as auto-submitted SAML forms).
    /// </summary>
    private string? _ssoHost;

    /// <summary>When the current SSO flow started (UTC). Flows expire after
    /// <see cref="SsoFlowTimeout"/> so the chrome-less WebView can't render
    /// external sites indefinitely.</summary>
    private DateTime? _ssoFlowStartedAt;

    /// <summary>How long an SSO flow may run before external navigation is cut off.</summary>
    private static readonly TimeSpan SsoFlowTimeout = TimeSpan.FromMinutes(10);

    private bool SsoFlowActive => _ssoHost != null;

    private bool SsoFlowExpired =>
        _ssoFlowStartedAt is { } started && DateTime.UtcNow - started > SsoFlowTimeout;

    /// <summary>Resets SSO state. Called on window hide, navigation errors, and
    /// when navigation returns to the Fleet host from an SSO flow.</summary>
    private void ResetSsoFlow()
    {
        _ssoHost = null;
        _ssoFlowStartedAt = null;
    }

    /// <summary>Case-insensitive check against the Fleet server host. All
    /// navigation-policy decisions go through this one comparison.</summary>
    private bool IsFleetHost(string? host) =>
        host != null && string.Equals(host, _fleetHost, StringComparison.OrdinalIgnoreCase);

    /// <summary>Case-insensitive check against the pinned SSO/IdP host.</summary>
    private bool IsCurrentSsoHost(string? host) =>
        host != null && string.Equals(host, _ssoHost, StringComparison.OrdinalIgnoreCase);

    public MainWindow()
    {
        InitializeComponent();
        Title = WindowTitle;
        CommandBindings.Add(new CommandBinding(NavigationCommands.Refresh, (_, _) => ReloadCurrent()));
        InputBindings.Add(new KeyBinding(NavigationCommands.Refresh, Key.F5, ModifierKeys.None));
        InputBindings.Add(new KeyBinding(NavigationCommands.Refresh, Key.R, ModifierKeys.Control));
    }

    /// <summary>Start loading the URL in the WebView without showing the window.</summary>
    public void Preload(Uri url)
    {
        _fleetHost = url.Host;
        _homeUrl = url;
        _ = InitializeWebViewAsync();
    }

    /// <summary>Reload whatever page is currently shown.</summary>
    public void ReloadCurrent()
    {
        if (_coreReady)
        {
            WebView.CoreWebView2.Reload();
        }
    }

    /// <summary>Navigate the WebView to a new URL (e.g. after token refresh).</summary>
    public void Reload(Uri url)
    {
        _fleetHost = url.Host;
        _homeUrl = url;
        if (_coreReady)
        {
            WebView.CoreWebView2.Navigate(url.ToString());
        }
        // else: InitializeWebViewAsync navigates to the latest _homeUrl once the
        // core is up, so a Reload during initialization is not lost.
    }

    /// <summary>
    /// Queue JavaScript to run once, the next time a Fleet-host page finishes loading.
    /// Set this *before* calling <see cref="Preload"/> or <see cref="Reload"/>.
    /// </summary>
    public void RunOnNextLoad(string js)
    {
        _pendingPostLoadJs = js;
    }

    /// <summary>Update the taskbar overlay icon based on the failing-policy count.
    /// Set unconditionally each poll (like the macOS Dock badge) so the overlay
    /// survives taskbar recreation, e.g. an Explorer restart.</summary>
    public void SetBadge(int failingPoliciesCount)
    {
        if (TaskbarItemInfo == null)
        {
            TaskbarItemInfo = new System.Windows.Shell.TaskbarItemInfo();
        }
        if (failingPoliciesCount <= 0)
        {
            TaskbarItemInfo.Overlay = null;
            TaskbarItemInfo.Description = WindowTitle;
        }
        else
        {
            TaskbarItemInfo.Overlay = TaskbarBadge.Create(failingPoliciesCount);
            TaskbarItemInfo.Description = failingPoliciesCount == 1
                ? "1 failing policy"
                : $"{failingPoliciesCount} failing policies";
        }
    }

    // ---- WebView2 init -------------------------------------------------------

    private async Task InitializeWebViewAsync()
    {
        try
        {
            // Per-launch user data folder under %LOCALAPPDATA% so cookies/cache
            // don't persist between sessions (mirrors WKWebView .nonPersistent()).
            var localAppData = Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData);
            var webViewRoot = Path.Combine(localAppData, "FleetDesktop", "WebView2");
            var userDataFolder = Path.Combine(webViewRoot, Guid.NewGuid().ToString("N"));
            Directory.CreateDirectory(userDataFolder);

            // Earlier launches leave their per-launch folders behind (the browser
            // process holds them until it exits, so they can't be deleted on
            // shutdown). Sweep stale siblings in the background; folders still in
            // use simply fail to delete and are skipped.
            _ = Task.Run(() => CleanupStaleUserDataFolders(webViewRoot, userDataFolder));

            var env = await CoreWebView2Environment.CreateAsync(
                browserExecutableFolder: null,
                userDataFolder: userDataFolder).ConfigureAwait(true);
            await WebView.EnsureCoreWebView2Async(env).ConfigureAwait(true);

            var core = WebView.CoreWebView2;
            core.Settings.AreDefaultContextMenusEnabled = true;
            core.Settings.AreDevToolsEnabled = false;
            core.Settings.IsStatusBarEnabled = false;
            core.Settings.IsZoomControlEnabled = true;
            core.Settings.AreHostObjectsAllowed = false;
            core.Settings.IsScriptEnabled = true;
            core.Settings.IsBuiltInErrorPageEnabled = true;
            // The profile is throwaway — never offer to save passwords or autofill data.
            core.Settings.IsPasswordAutosaveEnabled = false;
            core.Settings.IsGeneralAutofillEnabled = false;

            core.NavigationStarting += OnNavigationStarting;
            core.NavigationCompleted += OnNavigationCompleted;
            core.NewWindowRequested += OnNewWindowRequested;
            core.WebResourceResponseReceived += OnWebResourceResponseReceived;
            core.DownloadStarting += OnDownloadStarting;

            _coreReady = true;
            // Navigate to the *current* home URL — Reload() may have replaced it
            // while the core was initializing (e.g. a fleet:// deep link that
            // arrived during startup).
            if (_homeUrl != null)
            {
                core.Navigate(_homeUrl.ToString());
            }
        }
        catch (Exception ex)
        {
            Debug.WriteLine($"Fleet Desktop: WebView2 init failed: {ex}");
            MessageBox.Show(
                "Fleet Desktop requires the Microsoft Edge WebView2 Runtime to be installed.\n\n" +
                "If this Windows install is older than Windows 11, please install the WebView2 Evergreen Runtime " +
                "from https://developer.microsoft.com/microsoft-edge/webview2/ and try again.",
                WindowTitle,
                MessageBoxButton.OK,
                MessageBoxImage.Error);
            Application.Current.Shutdown(2);
        }
    }

    // ---- Navigation handlers -------------------------------------------------

    private void OnNavigationStarting(object? sender, CoreWebView2NavigationStartingEventArgs e)
    {
        // Track the top-level navigation target so OnWebResourceResponseReceived can match
        // the main-document response (WebView2 doesn't surface ResourceContext on that event).
        _pendingNavigationUri = e.Uri;

        if (!Uri.TryCreate(e.Uri, UriKind.Absolute, out var uri))
        {
            e.Cancel = true;
            return;
        }
        var scheme = uri.Scheme?.ToLowerInvariant() ?? "";

        // Same-host navigations and about: are always allowed.
        if (IsFleetHost(uri.Host)) return;
        if (scheme == "about") return;

        // During an active SSO flow, keep external IdP redirects in the WebView so
        // the chain completes in-app — but only over HTTPS, only while the flow is
        // fresh, and only on the current IdP host. Hops to a *new* external host
        // are allowed via server redirects or scripted navigations (multi-host IdP
        // chains, auto-submitted SAML forms); user link clicks to unrelated hosts
        // open in the default browser so the chrome-less WebView can't be steered
        // to arbitrary sites.
        if (SsoFlowActive)
        {
            if (scheme != "https" || SsoFlowExpired)
            {
                // Flow over (expired or degraded to non-HTTPS). Don't just cancel —
                // that would strand the WebView on the IdP page; return home.
                ResetSsoFlow();
                e.Cancel = true;
                NavigateHome();
                return;
            }
            if (IsCurrentSsoHost(uri.Host)) return;
            if (e.IsRedirected || !e.IsUserInitiated)
            {
                _ssoHost = uri.Host.ToLowerInvariant();
                return;
            }
            e.Cancel = true;
            OpenExternal(uri);
            return;
        }

        // Detect SSO: the WebView is showing a Fleet page (no flow active) and
        // something other than a user link click — a server redirect or scripted
        // navigation — is sending it to an external HTTPS host. Start the SSO
        // flow there. Requiring the committed document to be the Fleet page
        // mirrors the macOS sourceFrame check: without it, a page left behind by
        // an expired flow could restart the flow forever, making the 10-minute
        // cap meaningless.
        if (scheme == "https" && (e.IsRedirected || !e.IsUserInitiated) &&
            IsFleetHost(GetCurrentHost()))
        {
            _ssoHost = uri.Host.ToLowerInvariant();
            _ssoFlowStartedAt = DateTime.UtcNow;
            return;
        }

        // External link from a Fleet page: open in default browser if safe. But if
        // the WebView is stranded on an external page with no active flow (an
        // expired or abandoned SSO), navigate home instead — otherwise every
        // scripted retry on the stranded page would pop another browser tab.
        e.Cancel = true;
        if (IsFleetHost(GetCurrentHost()))
        {
            OpenExternal(uri);
        }
        else
        {
            NavigateHome();
        }
    }

    /// <summary>Returns the WebView to the Fleet device page. Used to recover
    /// from a stranded state (expired/abandoned SSO flow on an external page).</summary>
    private void NavigateHome()
    {
        if (_coreReady && _homeUrl != null)
        {
            WebView.CoreWebView2.Navigate(_homeUrl.ToString());
        }
    }

    private async void OnNavigationCompleted(object? sender, CoreWebView2NavigationCompletedEventArgs e)
    {
        var status = _pendingStatusCode;
        _pendingStatusCode = null;

        if (!e.IsSuccess)
        {
            ResetSsoFlow();
            NavigationError?.Invoke();
            return;
        }

        if (status is 401 or 403)
        {
            ResetSsoFlow();
            NavigationError?.Invoke();
            return;
        }

        Title = WindowTitle;
        if (LoadingOverlay.Visibility == Visibility.Visible)
        {
            await FadeOutAsync(LoadingOverlay, TimeSpan.FromMilliseconds(250));
        }

        var onFleetHost = IsFleetHost(GetCurrentHost());

        // If an SSO flow was active and we've finished loading a Fleet-host page,
        // the SSO callback is complete — reset the flow.
        if (SsoFlowActive && onFleetHost)
        {
            ResetSsoFlow();
        }

        await CheckPageForErrorsAsync();

        // Only run queued JS on Fleet-host pages — avoids injecting into IdP pages
        // during SSO redirects and avoids consuming the slot on an intermediate
        // redirect before the real target finishes loading.
        if (_pendingPostLoadJs is { } js && onFleetHost)
        {
            _pendingPostLoadJs = null;
            try
            {
                await WebView.CoreWebView2.ExecuteScriptAsync(js);
            }
            catch
            {
                // Best-effort — the page simply won't get the click.
            }
        }
    }

    /// <summary>Host of the document currently loaded in the WebView, or null.</summary>
    private string? GetCurrentHost()
    {
        var source = _coreReady ? WebView.CoreWebView2.Source : null;
        return Uri.TryCreate(source, UriKind.Absolute, out var uri) ? uri.Host : null;
    }

    /// <summary>
    /// Captures HTTP status codes for the top-level document so we can detect 401/403 on token expiry.
    /// </summary>
    private void OnWebResourceResponseReceived(object? sender, CoreWebView2WebResourceResponseReceivedEventArgs e)
    {
        try
        {
            if (e.Request?.Method != "GET") return;
            var uri = e.Request?.Uri;
            if (string.IsNullOrEmpty(uri)) return;
            // Filter to the main document: match the URI against the pending top-level navigation.
            // Subresources (CSS/JS/images) have different URIs so they're filtered out automatically.
            if (!string.Equals(uri, _pendingNavigationUri, StringComparison.Ordinal)) return;
            _pendingStatusCode = e.Response?.StatusCode;
        }
        catch
        {
            // Ignore malformed events.
        }
    }

    /// <summary>
    /// Fleet returns HTTP 200 with error HTML when a token expires. Check for a combination
    /// of error strings (2 of 3) to reduce false positives.
    /// </summary>
    private async Task CheckPageForErrorsAsync()
    {
        const string js = @"
            (function() {
                var body = document.body ? document.body.innerText : '';
                var errors = 0;
                if (body.indexOf('Something went wrong') !== -1) errors++;
                if (body.indexOf('Error loading software') !== -1) errors++;
                if (body.indexOf('Please contact your IT admin') !== -1) errors++;
                return errors >= 2 ? 'error' : 'ok';
            })();
        ";
        try
        {
            var result = await WebView.CoreWebView2.ExecuteScriptAsync(js);
            if (result != null && result.Contains("\"error\"", StringComparison.Ordinal))
            {
                NavigationError?.Invoke();
            }
        }
        catch
        {
            // Ignore — best-effort detection.
        }
    }

    private void OnNewWindowRequested(object? sender, CoreWebView2NewWindowRequestedEventArgs e)
    {
        // target="_blank" / window.open: route same-host (or the current SSO host,
        // while the flow is fresh) to this view, everything else to the default browser.
        e.Handled = true;
        if (!Uri.TryCreate(e.Uri, UriKind.Absolute, out var uri)) return;
        if (IsFleetHost(uri.Host) ||
            (SsoFlowActive && !SsoFlowExpired && IsCurrentSsoHost(uri.Host)))
        {
            WebView.CoreWebView2.Navigate(uri.ToString());
        }
        else
        {
            OpenExternal(uri);
        }
    }

    private void OnDownloadStarting(object? sender, CoreWebView2DownloadStartingEventArgs e)
    {
        try
        {
            // Rehome the file under the user's Downloads folder with a deduplicated
            // name. Only the final path component of WebView2's suggested path is
            // kept, so a server-supplied filename can't steer the file elsewhere.
            var downloads = Path.Combine(
                Environment.GetFolderPath(Environment.SpecialFolder.UserProfile),
                "Downloads");
            if (!Directory.Exists(downloads))
            {
                downloads = Environment.GetFolderPath(Environment.SpecialFolder.MyDocuments);
            }

            var fileName = Path.GetFileName(e.ResultFilePath);
            if (string.IsNullOrEmpty(fileName))
            {
                fileName = "download";
            }
            var dest = Path.Combine(downloads, fileName);
            var counter = 1;
            var baseName = Path.GetFileNameWithoutExtension(fileName);
            var fileExt = Path.GetExtension(fileName);
            while (File.Exists(dest) && counter < 1000)
            {
                dest = Path.Combine(downloads,
                    string.IsNullOrEmpty(fileExt)
                        ? $"{baseName} ({counter})"
                        : $"{baseName} ({counter}){fileExt}");
                counter++;
            }
            e.ResultFilePath = dest;
        }
        catch
        {
            // Fall back to WebView2's default download behavior.
        }
    }

    // ---- External URL handling ----------------------------------------------

    private static void OpenExternal(Uri uri)
    {
        var scheme = uri.Scheme?.ToLowerInvariant() ?? "";
        if (!AllowedExternalSchemes.Contains(scheme)) return;
        try
        {
            Process.Start(new ProcessStartInfo
            {
                FileName = uri.ToString(),
                UseShellExecute = true,
            });
        }
        catch
        {
            // Best-effort; if the OS can't open the URL, we don't bubble up.
        }
    }

    // ---- Show/Close hooks ----------------------------------------------------

    protected override void OnClosing(System.ComponentModel.CancelEventArgs e)
    {
        // Hide instead of close so the WebView stays alive and reopening is instant.
        e.Cancel = true;
        ResetSsoFlow();
        _pendingPostLoadJs = null;
        Hide();
    }

    /// <summary>
    /// Best-effort delete of per-launch WebView2 profile folders left behind by
    /// earlier runs. Folders still held open by a live browser process are left
    /// untouched; they'll be swept on a later launch.
    /// </summary>
    private static void CleanupStaleUserDataFolders(string root, string keep)
    {
        try
        {
            foreach (var dir in Directory.EnumerateDirectories(root))
            {
                if (string.Equals(dir, keep, StringComparison.OrdinalIgnoreCase)) continue;
                try
                {
                    // A recursive delete is not atomic — it would strip unlocked
                    // files out of a profile a live instance (same user, another
                    // session) is still using before failing on a locked one.
                    // Renaming the directory first fails outright while any file
                    // inside is open, so in-use profiles are never touched.
                    var tombstone = dir + ".stale";
                    Directory.Move(dir, tombstone);
                    Directory.Delete(tombstone, recursive: true);
                }
                catch
                {
                    // In use or already gone — skip.
                }
            }
        }
        catch
        {
            // Root missing or unreadable — nothing to sweep.
        }
    }

    public new void Show()
    {
        if (Visibility != Visibility.Visible)
        {
            base.Show();
        }
        if (WindowState == WindowState.Minimized) WindowState = WindowState.Normal;
        Activate();
        // Brief Topmost flicker forces the window to the foreground — common WPF pattern
        // for bringing a hidden window forward without leaving it pinned on top.
        Topmost = true;
        Topmost = false;
        Focus();

        WindowShown?.Invoke();
    }

    private static async Task FadeOutAsync(UIElement element, TimeSpan duration)
    {
        var anim = new System.Windows.Media.Animation.DoubleAnimation
        {
            From = 1.0,
            To = 0.0,
            Duration = duration,
        };
        var tcs = new TaskCompletionSource<bool>();
        anim.Completed += (_, _) =>
        {
            element.Visibility = Visibility.Collapsed;
            element.Opacity = 1.0;
            tcs.TrySetResult(true);
        };
        element.BeginAnimation(UIElement.OpacityProperty, anim);
        await tcs.Task;
    }
}
