using System;
using System.Diagnostics;
using System.IO;
using System.IO.Pipes;
using System.Security.Principal;
using System.Text;
using System.Threading;
using System.Threading.Tasks;

namespace FleetDesktop;

/// <summary>
/// Enforces a single running instance per user.
///
/// First-instance path: acquires a named mutex and starts a named-pipe server.
/// Later-instance path: connects to the pipe, sends the incoming fleet:// URL —
/// or a "reopen" request when launched with no URL — then exits; the running
/// app handles it.
///
/// The mutex uses the Local\ (per-session) namespace. Pipe names are global to
/// the machine, so the current user's SID is baked into the name — two users on
/// the same machine each get their own pipe — and both ends use
/// PipeOptions.CurrentUserOnly so another local user can neither connect to our
/// server nor impersonate it with a squatted pipe.
/// </summary>
internal sealed class SingleInstance : IDisposable
{
    /// <summary>Message a later instance sends when launched without a URL:
    /// the primary should show its window.</summary>
    public const string ReopenMessage = "reopen";

    private readonly string _name;
    private Mutex? _mutex;
    private bool _mutexOwned;
    private CancellationTokenSource? _serverCts;

    /// <summary>Raised with each message received from a later instance —
    /// either a fleet:// URL or <see cref="ReopenMessage"/>.</summary>
    public event Action<string>? OnMessageReceived;

    public SingleInstance(string name)
    {
        _name = name;
    }

    /// <summary>
    /// Returns true if this is the first instance (we own the mutex and the pipe server is running).
    /// Returns false if another instance is running; in that case <paramref name="urlToForward"/>
    /// has been sent to the running instance — or, when it is null and
    /// <paramref name="requestReopenIfNoUrl"/> is set, a reopen request so the primary
    /// shows its window. A hidden launch (autostart) passes false so it never pops UI.
    /// </summary>
    public bool AcquireOrForward(string? urlToForward, bool requestReopenIfNoUrl)
    {
        _mutex = new Mutex(initiallyOwned: false, name: $"Local\\{_name}.mutex");
        try
        {
            _mutexOwned = _mutex.WaitOne(0);
        }
        catch (AbandonedMutexException)
        {
            // Previous owner crashed — we now own it.
            _mutexOwned = true;
        }

        if (!_mutexOwned)
        {
            if (!string.IsNullOrEmpty(urlToForward))
            {
                TryForwardMessage(urlToForward);
            }
            else if (requestReopenIfNoUrl)
            {
                TryForwardMessage(ReopenMessage);
            }
            return false;
        }

        StartPipeServer();
        return true;
    }

    private void StartPipeServer()
    {
        _serverCts = new CancellationTokenSource();
        var ct = _serverCts.Token;
        _ = Task.Run(() => RunPipeServerLoop(ct), ct);
    }

    private async Task RunPipeServerLoop(CancellationToken ct)
    {
        var pipeName = PipeName(_name);
        while (!ct.IsCancellationRequested)
        {
            try
            {
                using var server = new NamedPipeServerStream(
                    pipeName,
                    PipeDirection.In,
                    maxNumberOfServerInstances: 1,
                    PipeTransmissionMode.Byte,
                    PipeOptions.Asynchronous | PipeOptions.CurrentUserOnly);
                await server.WaitForConnectionAsync(ct).ConfigureAwait(false);
                using var reader = new StreamReader(server, Encoding.UTF8);
                var message = (await reader.ReadToEndAsync(ct).ConfigureAwait(false)).Trim();
                if (message.Length > 0)
                {
                    OnMessageReceived?.Invoke(message);
                }
            }
            catch (OperationCanceledException)
            {
                return;
            }
            catch
            {
                // Recreate the pipe after a beat — a tight retry loop would spin
                // the CPU if creation fails persistently (e.g. the name is squatted).
                try
                {
                    await Task.Delay(TimeSpan.FromSeconds(1), ct).ConfigureAwait(false);
                }
                catch (OperationCanceledException)
                {
                    return;
                }
            }
        }
    }

    private void TryForwardMessage(string message)
    {
        try
        {
            using var client = new NamedPipeClientStream(
                ".", PipeName(_name), PipeDirection.Out, PipeOptions.CurrentUserOnly);
            client.Connect(2000);
            using var writer = new StreamWriter(client, new UTF8Encoding(encoderShouldEmitUTF8Identifier: false));
            writer.Write(message);
            writer.Flush();
        }
        catch
        {
            // Best-effort — if the other instance can't be reached, the message is dropped.
        }
    }

    private static string PipeName(string name)
    {
        // Pipe names share one machine-global namespace. Scope by user SID and
        // session id: different users get their own pipe, and so do the same
        // user's concurrent console/RDP sessions — each session has its own
        // Local\ mutex and therefore its own primary instance, which would
        // otherwise fight over a single pipe.
        string sid;
        try
        {
            sid = WindowsIdentity.GetCurrent().User?.Value ?? "default";
        }
        catch
        {
            sid = "default";
        }
        int sessionId;
        try
        {
            sessionId = Process.GetCurrentProcess().SessionId;
        }
        catch
        {
            sessionId = 0;
        }
        return $"{name}.{sid}.s{sessionId}.pipe";
    }

    public void Dispose()
    {
        _serverCts?.Cancel();
        _serverCts?.Dispose();
        _serverCts = null;

        if (_mutex != null)
        {
            if (_mutexOwned)
            {
                try { _mutex.ReleaseMutex(); } catch { }
                _mutexOwned = false;
            }
            _mutex.Dispose();
            _mutex = null;
        }
    }
}
