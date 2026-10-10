using System;
using System.IO;
using System.Net.Http;
using System.Text.Json;
using System.Text.Json.Serialization;
using System.Text.RegularExpressions;
using System.Threading;
using System.Threading.Tasks;
using System.Web;
using System.Windows;
using System.Windows.Threading;
using Microsoft.Win32;

namespace FleetDesktop;

/// <summary>
/// Core service: resolves config, reads the device token, owns the browser window,
/// polls for token rotation, and updates the taskbar overlay badge.
///
/// Port of FleetService.swift from the macOS app, adapted to WPF idioms.
/// Mutable state is protected by <see cref="_stateLock"/>.
/// </summary>
internal sealed class FleetService : IDisposable
{
    public const string WindowTitle = "Fleet Desktop";

    /// <summary>Pages accessible via fleet:// URLs.</summary>
    private static readonly HashSet<string> ValidPages =
        new(StringComparer.OrdinalIgnoreCase) { "self-service", "policies", "software" };

    private static readonly TimeSpan TokenRefreshInterval = TimeSpan.FromSeconds(60);
    private static readonly TimeSpan TokenRetryDelay = TimeSpan.FromSeconds(5);
    private const int MaxRetryAttempts = 3;

    private static readonly HttpClient Http = new(new SocketsHttpHandler
    {
        PooledConnectionLifetime = TimeSpan.FromMinutes(2),
    })
    {
        Timeout = TimeSpan.FromSeconds(30),
    };

    private readonly object _stateLock = new();
    private readonly string _orbitRoot;
    private readonly string _tokenFile;
    private readonly string? _fleetUrlFromService;

    private MainWindow? _browserWindow;
    private DispatcherTimer? _refreshTimer;

    // Mutable state — guard with _stateLock.
    private string? _baseUrl;
    private string? _currentToken;
    private bool _isSettingUp;
    private int _retryCount;
    private string? _pendingPage;
    private bool _pendingRefetch;
    private bool _pendingUpdateAll;
    private bool _pendingInstallAll;
    private string? _pendingInstallAllCategoryId;
    private bool _userRequestedFleetUi;

    /// <summary>Most recent failing_policies_count from the desktop API. Guarded by _stateLock.</summary>
    private int? _lastBadgeCount;

    /// <summary>
    /// The failing_policies_count reflected by the currently loaded web page. Compared to
    /// <see cref="_lastBadgeCount"/> when the window is shown to detect a stale Policies tab
    /// (e.g. the count dropped to 0 while the window was closed). Guarded by _stateLock.
    /// </summary>
    private int? _pageBadgeCount;

    // Main-thread-only.
    private bool _hidden;

    public FleetService()
    {
        // On Windows, the Fleet URL and root-dir live as command-line arguments
        // baked into the "Fleet osquery" service's ImagePath registry value by
        // fleetd's own WiX installer. (Unlike macOS, there's no fleet_url.txt —
        // that file is written only on darwin from an Apple config profile.)
        var serviceArgs = ReadOrbitServiceArgs();
        _fleetUrlFromService = serviceArgs != null ? ExtractArg(serviceArgs, "fleet-url") : null;

        _orbitRoot = ResolveOrbitRoot(serviceArgs);
        _tokenFile = Path.Combine(_orbitRoot, "identifier");
    }

    private static string ResolveOrbitRoot(string? serviceArgs)
    {
        var fromEnv = Environment.GetEnvironmentVariable("ORBIT_ROOT_DIR");
        if (!string.IsNullOrWhiteSpace(fromEnv))
        {
            return fromEnv.Trim();
        }
        if (serviceArgs != null)
        {
            var fromService = ExtractArg(serviceArgs, "root-dir");
            if (!string.IsNullOrWhiteSpace(fromService))
            {
                // fleetd's WiX template uses [ORBITROOT]. → trailing "\." which Path.Combine
                // tolerates, but trim it so error messages show a clean path.
                return fromService.TrimEnd('.', '\\', '/').TrimEnd('\\', '/');
            }
        }
        // Last-resort default install path.
        var programFiles = Environment.GetFolderPath(Environment.SpecialFolder.ProgramFiles);
        if (string.IsNullOrEmpty(programFiles))
        {
            programFiles = @"C:\Program Files";
        }
        return Path.Combine(programFiles, "Orbit");
    }

    /// <summary>
    /// Reads HKLM\SYSTEM\CurrentControlSet\Services\Fleet osquery\ImagePath.
    /// Returns the full command line (exe path + args) or null if the service
    /// isn't installed / can't be read.
    /// </summary>
    private static string? ReadOrbitServiceArgs()
    {
        try
        {
            using var key = Registry.LocalMachine.OpenSubKey(
                @"SYSTEM\CurrentControlSet\Services\Fleet osquery");
            // GetValue with DoNotExpandEnvironmentNames preserves any %VAR% tokens; the
            // ImagePath for fleetd is typically a plain string but be safe.
            return key?.GetValue("ImagePath", null,
                RegistryValueOptions.DoNotExpandEnvironmentNames) as string;
        }
        catch
        {
            return null;
        }
    }

    /// <summary>
    /// Extracts a "--name" argument value from a command-line string. Supports both
    /// quoted (--name "value") and unquoted (--name value, --name=value) forms.
    /// </summary>
    private static string? ExtractArg(string commandLine, string name)
    {
        var escaped = Regex.Escape(name);
        // --name "value with spaces"
        var quoted = Regex.Match(commandLine, $@"--{escaped}[=\s]+""([^""]+)""");
        if (quoted.Success) return quoted.Groups[1].Value;
        // --name value-without-spaces  or  --name=value
        var bare = Regex.Match(commandLine, $@"--{escaped}[=\s]+(\S+)");
        return bare.Success ? bare.Groups[1].Value : null;
    }

    // ---- Public entry points -------------------------------------------------

    /// <summary>
    /// Initial run. Resolves config, creates the browser window, and either shows
    /// or defers presentation depending on <paramref name="hidden"/>. UI thread only.
    /// </summary>
    public void Run(string? initialUrl, bool hidden)
    {
        _hidden = hidden;

        // A fleet:// launch goes through the same dispatch as a forwarded URL —
        // one host-matching table for both cold start and running-app delivery.
        // It queues the requested action and kicks setup via ShowWindow.
        if (!string.IsNullOrEmpty(initialUrl))
        {
            HandleFleetUrl(initialUrl);
            if (Uri.TryCreate(initialUrl, UriKind.Absolute, out var url) &&
                string.Equals(url.Scheme, "fleet", StringComparison.OrdinalIgnoreCase))
            {
                return;
            }
            // Unparseable URL: fall through to a plain launch.
        }

        // Plain launch — resolve config off the UI thread, then bounce back to
        // create the window. Doesn't mark the UI as user-requested, so a
        // --hidden launch stays hidden.
        bool shouldKick;
        lock (_stateLock)
        {
            shouldKick = !_isSettingUp;
            if (shouldKick) _isSettingUp = true;
        }
        if (shouldKick) Task.Run(Setup);
    }

    /// <summary>Called by App when a forwarded fleet:// URL arrives (named pipe).</summary>
    public void HandleFleetUrl(string urlString)
    {
        if (!Uri.TryCreate(urlString, UriKind.Absolute, out var url) ||
            !string.Equals(url.Scheme, "fleet", StringComparison.OrdinalIgnoreCase))
        {
            return;
        }

        var host = url.Host?.ToLowerInvariant() ?? string.Empty;
        // The window buffers navigations and queued JS internally until its
        // WebView2 core finishes initializing, so it counts as ready the moment
        // it exists — deep links arriving mid-startup are not dropped.
        var browserReady = _browserWindow != null;

        if (!browserReady)
        {
            lock (_stateLock) { _userRequestedFleetUi = true; }
        }

        if (host == "refetch")
        {
            bool hasConfig;
            lock (_stateLock) { hasConfig = _baseUrl != null; }
            if (hasConfig)
            {
                _ = PerformRefetchAsync();
            }
            else
            {
                lock (_stateLock) { _pendingRefetch = true; }
            }
            ShowWindow();
            return;
        }

        // fleet://update_all (or fleet://update-all) — open the self-service page
        // and click its "Update all" button via the WebView so the install logic
        // stays defined by Fleet's UI rather than duplicated here.
        if (host is "update_all" or "update-all")
        {
            if (browserReady)
            {
                TriggerUpdateAll();
            }
            else
            {
                lock (_stateLock) { _pendingUpdateAll = true; }
                ShowWindow();
            }
            return;
        }

        // fleet://install_all (or fleet://install-all) — open the self-service page
        // and click its "Install all" button, which opens Fleet's confirmation modal.
        // The user must explicitly confirm before anything installs. An optional
        // ?category_id=## first filters the page to that category so the install is
        // scoped to it, matching Fleet's own UI behavior.
        if (host is "install_all" or "install-all")
        {
            var categoryId = ExtractCategoryId(url);
            if (browserReady)
            {
                TriggerInstallAll(categoryId);
            }
            else
            {
                lock (_stateLock)
                {
                    _pendingInstallAll = true;
                    _pendingInstallAllCategoryId = categoryId;
                }
                ShowWindow();
            }
            return;
        }

        var page = ValidPages.Contains(host) ? host : null;

        if (browserReady && _browserWindow != null)
        {
            if (page != null && TryBuildDeviceUrl(page, out var target))
            {
                lock (_stateLock) { _pageBadgeCount = _lastBadgeCount; }
                _browserWindow.Reload(target!);
            }
            _browserWindow.Show();
            return;
        }

        lock (_stateLock) { _pendingPage = page; }
        // Setup is either running or done; if done, ShowWindow will re-present.
        ShowWindow();
    }

    /// <summary>Called when the user activates the app via taskbar/etc. with no specific URL.</summary>
    public void ShowWindow()
    {
        if (_browserWindow != null)
        {
            Application.Current.Dispatcher.BeginInvoke(() => _browserWindow.Show());
            return;
        }

        // Setup may still be running — pending state will get consumed when it completes.
        bool shouldKick;
        lock (_stateLock)
        {
            shouldKick = !_isSettingUp;
            if (shouldKick) _isSettingUp = true;
            _userRequestedFleetUi = true;
        }
        if (shouldKick)
        {
            Task.Run(Setup);
        }
    }

    /// <summary>Reload the current page (View → Reload, F5, Ctrl+R).</summary>
    public void ReloadCurrentPage()
    {
        _browserWindow?.ReloadCurrent();
    }

    /// <summary>
    /// Navigates to the self-service page and clicks its "Update all" button,
    /// reusing the Fleet UI's own filter/install logic. Called when fleet://update_all
    /// arrives after the browser has been set up. UI thread only.
    /// </summary>
    private void TriggerUpdateAll()
    {
        if (_browserWindow == null || !TryBuildDeviceUrl("self-service", out var target)) return;
        // Sync the page-state count: the navigation below brings the page current,
        // and a stale count would make ReloadIfPoliciesStale (fired by Show) issue
        // a competing reload that cancels this navigation.
        lock (_stateLock) { _pageBadgeCount = _lastBadgeCount; }
        _browserWindow.RunOnNextLoad(UpdateAllJs);
        _browserWindow.Reload(target!);
        _browserWindow.Show();
    }

    /// <summary>
    /// Navigates to the self-service page (optionally filtered to a category) and
    /// clicks its "Install all" button, which opens Fleet's confirmation modal for
    /// the user to accept. Called when fleet://install_all arrives after the browser
    /// has been set up. UI thread only.
    /// </summary>
    private void TriggerInstallAll(string? categoryId)
    {
        if (_browserWindow == null || !TryBuildDeviceUrl("self-service", categoryId, out var target)) return;
        // See TriggerUpdateAll for why the page-state count is synced first.
        lock (_stateLock) { _pageBadgeCount = _lastBadgeCount; }
        _browserWindow.RunOnNextLoad(InstallAllJs);
        _browserWindow.Reload(target!);
        _browserWindow.Show();
    }

    /// <summary>
    /// JS injected into the self-service page to click its "Update all" button.
    /// Retries for a few seconds because the React UI mounts asynchronously after
    /// the navigation completes. Matching on visible button text keeps the install
    /// logic owned by Fleet's UI rather than duplicated in this app.
    /// </summary>
    private const string UpdateAllJs = """
        (function() {
            var attempts = 0;
            var maxAttempts = 60; // ~30s at 500ms
            function tryClick() {
                var btns = document.querySelectorAll('button');
                for (var i = 0; i < btns.length; i++) {
                    var label = (btns[i].textContent || '').trim();
                    if (label === 'Update all' && !btns[i].disabled) {
                        btns[i].click();
                        return;
                    }
                }
                if (++attempts < maxAttempts) {
                    setTimeout(tryClick, 500);
                }
            }
            tryClick();
        })();
        """;

    /// <summary>
    /// JS injected into the self-service page to click its "Install all" button.
    /// The trigger button is labeled "Install all (N)" (count in parentheses);
    /// clicking it opens Fleet's confirmation modal. We intentionally stop there —
    /// the user must explicitly confirm in the modal before anything installs, so
    /// the deep link never starts installs without acknowledgment. If the trigger
    /// is disabled (nothing left to install) nothing happens, which is the desired
    /// outcome.
    /// </summary>
    private const string InstallAllJs = """
        (function() {
            var attempts = 0;
            var maxAttempts = 60; // ~30s at 500ms
            function tryClick() {
                var btns = document.querySelectorAll('button');
                for (var i = 0; i < btns.length; i++) {
                    var label = (btns[i].textContent || '').trim();
                    // Trigger button: "Install all (N)" — has a count in parentheses.
                    // Clicking it opens the confirmation modal; the user confirms.
                    if (label.indexOf('Install all') === 0 && label.indexOf('(') !== -1 && !btns[i].disabled) {
                        btns[i].click();
                        return;
                    }
                }
                if (++attempts < maxAttempts) {
                    setTimeout(tryClick, 500);
                }
            }
            tryClick();
        })();
        """;

    /// <summary>
    /// Extracts and validates the category_id query parameter from a fleet:// URL.
    /// Returns the numeric string when present and well-formed, otherwise null.
    /// First value wins, matching the macOS implementation. Validation keeps
    /// anything unexpected out of the URL we build (the Fleet UI parses
    /// category_id as an integer).
    /// </summary>
    private static string? ExtractCategoryId(Uri url)
    {
        var value = HttpUtility.ParseQueryString(url.Query).GetValues("category_id")?.FirstOrDefault();
        return !string.IsNullOrEmpty(value) && value.All(char.IsAsciiDigit) ? value : null;
    }

    public void Dispose()
    {
        _refreshTimer?.Stop();
        _refreshTimer = null;
    }

    // ---- Setup ---------------------------------------------------------------

    private void Setup()
    {
        // Double-guard re-entry. Checking for the window itself (not core
        // readiness) ensures a second Setup can never construct a second
        // MainWindow while the first one's WebView2 is still initializing.
        lock (_stateLock)
        {
            if (_browserWindow != null)
            {
                _isSettingUp = false;
                return;
            }
            // _isSettingUp may already be true (set by ShowWindow/Run); make sure it's true.
            _isSettingUp = true;
        }

        if (!ResolveConfig())
        {
            lock (_stateLock) { _isSettingUp = false; }
            return;
        }

        // Hop to the UI thread to create the window.
        Application.Current.Dispatcher.BeginInvoke(() =>
        {
            string page;
            bool refetch, updateAll, installAll;
            string? installAllCategoryId;
            lock (_stateLock)
            {
                page = _pendingPage ?? "self-service";
                _pendingPage = null;
                refetch = _pendingRefetch;
                _pendingRefetch = false;
                updateAll = _pendingUpdateAll;
                _pendingUpdateAll = false;
                installAll = _pendingInstallAll;
                _pendingInstallAll = false;
                installAllCategoryId = _pendingInstallAllCategoryId;
                _pendingInstallAllCategoryId = null;
            }

            if (refetch) _ = PerformRefetchAsync();

            // Update-all and install-all require the self-service page so the button is in the DOM.
            if (updateAll || installAll) page = "self-service";
            var categoryId = installAll ? installAllCategoryId : null;

            if (!TryBuildDeviceUrl(page, categoryId, out var url))
            {
                ShowFatalError("Unable to construct self-service URL. Check Fleet configuration.");
                return;
            }

            var window = new MainWindow();
            _browserWindow = window;
            window.NavigationError += OnNavigationError;
            window.WindowShown += OnWindowShown;
            if (updateAll)
            {
                window.RunOnNextLoad(UpdateAllJs);
            }
            else if (installAll)
            {
                window.RunOnNextLoad(InstallAllJs);
            }
            window.Preload(url!);

            StartRefreshTimer();

            // Defer the show decision one dispatcher turn so any pending fleet:// URL
            // forwarded via the named pipe is already in pending state.
            Application.Current.Dispatcher.BeginInvoke(() =>
            {
                bool userWants;
                lock (_stateLock)
                {
                    userWants = _userRequestedFleetUi;
                    _userRequestedFleetUi = false;
                }

                // Hidden launch with no user request: leave the window unshown.
                // The user gets it back via a fleet:// URL or a plain relaunch
                // (the second instance forwards a reopen request to us).
                if (!_hidden || userWants)
                {
                    window.Show();
                }

                lock (_stateLock) { _isSettingUp = false; }
            }, DispatcherPriority.Background);
        });
    }

    /// <summary>Reads the Fleet URL and device token. Returns true if both are valid.</summary>
    private bool ResolveConfig()
    {
        var fleetUrl = ReadFleetUrl();
        if (string.IsNullOrEmpty(fleetUrl))
        {
            ShowFatalError(
                "Fleet Desktop could not find the Fleet server URL.\n\n" +
                "Expected to read it from the \"Fleet osquery\" Windows service " +
                "(HKLM\\SYSTEM\\CurrentControlSet\\Services\\Fleet osquery\\ImagePath, " +
                "--fleet-url argument).\n\n" +
                "Ensure the Fleet orbit agent (fleetd) is installed and enrolled on this machine. " +
                "Contact your administrator if the problem persists.");
            return false;
        }

        // Require HTTPS — the device token is sent to this URL, and a misconfigured
        // http:// value would put it on the wire in cleartext.
        if (!Uri.TryCreate(fleetUrl, UriKind.Absolute, out var parsedFleetUrl) ||
            !string.Equals(parsedFleetUrl.Scheme, "https", StringComparison.OrdinalIgnoreCase))
        {
            ShowFatalError(
                "The configured Fleet URL must use HTTPS.\n\n" +
                $"Read: {fleetUrl}\n\n" +
                "Check the --fleet-url argument on the \"Fleet osquery\" service.");
            return false;
        }

        var token = ReadToken();
        if (string.IsNullOrEmpty(token))
        {
            ShowFatalError(
                $"Device token not found or could not be read at {_tokenFile}.\n\n" +
                "Ensure the Fleet orbit agent is enrolled and the identifier file exists.");
            return false;
        }

        lock (_stateLock)
        {
            _baseUrl = fleetUrl.TrimEnd('/');
            _currentToken = token;
        }
        return true;
    }

    // ---- Token refresh / badge polling ---------------------------------------

    private void StartRefreshTimer()
    {
        _refreshTimer?.Stop();
        _refreshTimer = new DispatcherTimer(DispatcherPriority.Background)
        {
            Interval = TokenRefreshInterval,
        };
        _refreshTimer.Tick += (_, _) =>
        {
            RefreshTokenIfNeeded();
            _ = FetchDesktopDataAsync();
        };
        _refreshTimer.Start();

        // Kick off an immediate badge fetch so the first update doesn't wait 60s.
        _ = FetchDesktopDataAsync();
    }

    private void RefreshTokenIfNeeded()
    {
        var newToken = ReadToken();
        if (string.IsNullOrEmpty(newToken) || _browserWindow == null) return;

        bool changed;
        lock (_stateLock)
        {
            changed = newToken != _currentToken;
            if (changed)
            {
                _currentToken = newToken;
                _retryCount = 0;
            }
        }
        if (!changed) return;

        if (TryBuildDeviceUrl("self-service", out var url))
        {
            lock (_stateLock) { _pageBadgeCount = _lastBadgeCount; }
            _browserWindow.Reload(url!);
        }
    }

    private void OnNavigationError()
    {
        string? oldToken;
        lock (_stateLock) { oldToken = _currentToken; }

        var newToken = ReadToken();
        if (!string.IsNullOrEmpty(newToken) && newToken != oldToken)
        {
            lock (_stateLock)
            {
                _currentToken = newToken;
                _retryCount = 0;
            }
            if (TryBuildDeviceUrl("self-service", out var url))
            {
                _browserWindow?.Reload(url!);
            }
            return;
        }

        bool retry;
        lock (_stateLock)
        {
            if (_retryCount < MaxRetryAttempts)
            {
                _retryCount++;
                retry = true;
            }
            else
            {
                _retryCount = 0;
                retry = false;
            }
        }
        if (!retry) return;

        _ = Task.Run(async () =>
        {
            await Task.Delay(TokenRetryDelay).ConfigureAwait(false);
            await Application.Current.Dispatcher.InvokeAsync(RefreshTokenIfNeeded);
        });
    }

    private void OnWindowShown()
    {
        RefreshTokenIfNeeded();
        ReloadIfPoliciesStale();
    }

    /// <summary>
    /// Reloads the web view when the badge count differs from what the currently
    /// loaded page is showing — e.g. the user closed the window with 1 failing
    /// policy, the badge later dropped to 0, and they're reopening to see the change.
    /// </summary>
    private void ReloadIfPoliciesStale()
    {
        int? current, rendered;
        lock (_stateLock)
        {
            current = _lastBadgeCount;
            rendered = _pageBadgeCount;
        }
        if (current == null || rendered == null || current == rendered || _browserWindow == null) return;
        lock (_stateLock) { _pageBadgeCount = current; }
        _browserWindow.ReloadCurrent();
    }

    private async Task PerformRefetchAsync()
    {
        if (!TryBuildApiUrl("refetch", out var url)) return;
        try
        {
            using var req = new HttpRequestMessage(HttpMethod.Post, url);
            using var resp = await Http.SendAsync(req).ConfigureAwait(false);
            if (!resp.IsSuccessStatusCode) return;
        }
        catch
        {
            return;
        }

        // After a refetch succeeds, sample the badge twice so policy changes
        // (e.g. an app install that flips a policy to passing) show up quickly.
        _ = Task.Run(async () =>
        {
            await Task.Delay(TimeSpan.FromSeconds(15)).ConfigureAwait(false);
            await FetchDesktopDataAsync().ConfigureAwait(false);
            await Task.Delay(TimeSpan.FromSeconds(15)).ConfigureAwait(false);
            await FetchDesktopDataAsync().ConfigureAwait(false);
        });
    }

    private async Task FetchDesktopDataAsync()
    {
        if (!TryBuildApiUrl("desktop", out var url)) return;
        try
        {
            using var resp = await Http.GetAsync(url).ConfigureAwait(false);
            if (!resp.IsSuccessStatusCode) return;
            var json = await resp.Content.ReadAsStringAsync().ConfigureAwait(false);
            var data = JsonSerializer.Deserialize<DesktopResponse>(json);
            if (data == null) return;

            lock (_stateLock)
            {
                _lastBadgeCount = data.FailingPoliciesCount;
                // On the first successful poll, seed the page-state count too —
                // the loaded web page reflects Fleet state at this same moment.
                _pageBadgeCount ??= data.FailingPoliciesCount;
            }

            await Application.Current.Dispatcher.InvokeAsync(() =>
            {
                _browserWindow?.SetBadge(data.FailingPoliciesCount);
            });
        }
        catch
        {
            // Network errors are silent — next tick will retry.
        }
    }

    private sealed class DesktopResponse
    {
        [JsonPropertyName("failing_policies_count")]
        public int FailingPoliciesCount { get; set; }
    }

    // ---- URL construction ----------------------------------------------------

    private bool TryBuildDeviceUrl(string page, out Uri? result) =>
        TryBuildDeviceUrl(page, null, out result);

    private bool TryBuildDeviceUrl(string page, string? categoryId, out Uri? result)
    {
        result = null;
        string? baseUrl, token;
        lock (_stateLock) { baseUrl = _baseUrl; token = _currentToken; }
        if (string.IsNullOrEmpty(baseUrl) || string.IsNullOrEmpty(token)) return false;
        var encodedToken = Uri.EscapeDataString(token);
        var encodedPage = Uri.EscapeDataString(page);
        var url = $"{baseUrl}/device/{encodedToken}/{encodedPage}";
        if (!string.IsNullOrEmpty(categoryId))
        {
            // Validated as numeric upstream; encoded anyway for defense in depth.
            url += $"?category_id={Uri.EscapeDataString(categoryId)}";
        }
        return Uri.TryCreate(url, UriKind.Absolute, out result);
    }

    private bool TryBuildApiUrl(string suffix, out Uri? result)
    {
        result = null;
        string? baseUrl, token;
        lock (_stateLock) { baseUrl = _baseUrl; token = _currentToken; }
        if (string.IsNullOrEmpty(baseUrl) || string.IsNullOrEmpty(token)) return false;
        var encodedToken = Uri.EscapeDataString(token);
        return Uri.TryCreate($"{baseUrl}/api/latest/fleet/device/{encodedToken}/{suffix}", UriKind.Absolute, out result);
    }

    // ---- Config / file reading ----------------------------------------------

    /// <summary>
    /// Returns the Fleet server URL parsed from the "Fleet osquery" service's
    /// ImagePath in the registry. Debug builds only: the FLEET_URL env var takes
    /// precedence for local testing. It is compiled out of Release builds because
    /// a user-writable environment variable must not be able to redirect the
    /// device token to an attacker-controlled host.
    /// </summary>
    private string? ReadFleetUrl()
    {
#if DEBUG
        var fromEnv = Environment.GetEnvironmentVariable("FLEET_URL");
        if (!string.IsNullOrWhiteSpace(fromEnv)) return fromEnv.Trim();
#endif
        return string.IsNullOrWhiteSpace(_fleetUrlFromService) ? null : _fleetUrlFromService;
    }

    /// <summary>
    /// Allowed device-token shape: alphanumerics plus - and _. Rejecting anything
    /// else keeps path separators and other URL metacharacters out of the device
    /// URLs built from the token.
    /// </summary>
    private static readonly Regex TokenPattern = new("^[A-Za-z0-9_-]+$", RegexOptions.Compiled);

    private string? ReadToken()
    {
        var token = ReadFileTrimmed(_tokenFile);
        return token != null && TokenPattern.IsMatch(token) ? token : null;
    }

    private static string? ReadFileTrimmed(string path)
    {
        try
        {
            if (!File.Exists(path)) return null;
            var contents = File.ReadAllText(path).Trim('\r', '\n', ' ', '\t');
            return string.IsNullOrEmpty(contents) ? null : contents;
        }
        catch
        {
            return null;
        }
    }

    // ---- Error display -------------------------------------------------------

    private static void ShowFatalError(string message)
    {
        void Work()
        {
            MessageBox.Show(message, WindowTitle, MessageBoxButton.OK, MessageBoxImage.Error);
            Application.Current.Shutdown(1);
        }
        if (Application.Current.Dispatcher.CheckAccess())
        {
            Work();
        }
        else
        {
            Application.Current.Dispatcher.Invoke(Work);
        }
    }
}
