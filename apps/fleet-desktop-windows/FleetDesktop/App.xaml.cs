using System;
using System.Threading.Tasks;
using System.Windows;

namespace FleetDesktop;

/// <summary>
/// WPF application entry. Mirrors the macOS AppDelegate:
/// - Enforces a single instance (mutex + named pipe to forward fleet:// URLs).
/// - Parses the command line for a fleet:// URL.
/// - Starts <see cref="FleetService"/>, which owns config, token rotation, and the browser window.
/// </summary>
public partial class App : Application
{
    private SingleInstance? _singleInstance;
    private FleetService? _fleetService;

    /// <summary>
    /// Whether the app was launched with a "hidden" flag (e.g. autostart on login).
    /// Equivalent to the macOS "open -j" / hidden login item case: window stays closed
    /// until the user explicitly foregrounds the app or a fleet:// URL arrives.
    /// </summary>
    public bool LaunchedHidden { get; private set; }

    protected override void OnStartup(StartupEventArgs e)
    {
        base.OnStartup(e);

        var (initialUrl, hidden) = ParseArgs(e.Args);
        LaunchedHidden = hidden;

        // v2: the pipe name gained a per-user SID suffix and the protocol gained
        // the "reopen" message. Versioning the channel name keeps a leftover
        // pre-upgrade instance from owning the mutex while newer binaries forward
        // messages to a pipe nobody is listening on.
        _singleInstance = new SingleInstance("FleetDesktop.Singleton.v2");
        if (!_singleInstance.AcquireOrForward(initialUrl, requestReopenIfNoUrl: !hidden))
        {
            // Another instance is running; we've forwarded the URL (or a reopen
            // request) to it. Exit quietly.
            Shutdown(0);
            return;
        }

        _singleInstance.OnMessageReceived += message =>
        {
            // Forwarded from a second-instance launch — dispatch on the UI thread.
            Dispatcher.BeginInvoke(() =>
            {
                if (message == SingleInstance.ReopenMessage)
                {
                    // Plain relaunch (Start Menu, taskbar) with no URL: show the window.
                    _fleetService?.ShowWindow();
                }
                else
                {
                    _fleetService?.HandleFleetUrl(message);
                }
            });
        };

        _fleetService = new FleetService();
        _fleetService.Run(initialUrl, hidden);
    }

    protected override void OnExit(ExitEventArgs e)
    {
        _fleetService?.Dispose();
        _singleInstance?.Dispose();
        base.OnExit(e);
    }

    private static (string? Url, bool Hidden) ParseArgs(string[] args)
    {
        string? url = null;
        bool hidden = false;
        foreach (var raw in args)
        {
            if (string.IsNullOrWhiteSpace(raw)) continue;
            var arg = raw.Trim();
            if (arg.Equals("--hidden", StringComparison.OrdinalIgnoreCase) ||
                arg.Equals("--minimized", StringComparison.OrdinalIgnoreCase) ||
                arg.Equals("/hidden", StringComparison.OrdinalIgnoreCase))
            {
                hidden = true;
                continue;
            }
            if (arg.StartsWith("fleet://", StringComparison.OrdinalIgnoreCase))
            {
                url = arg;
            }
        }
        return (url, hidden);
    }
}
