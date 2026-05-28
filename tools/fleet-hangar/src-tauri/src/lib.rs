mod db;
mod deps;
mod fleetctl;
mod git;
mod gitops;
mod perf;
mod processes;
mod settings;
mod shellpath;
mod tray;
mod troubleshoot;

use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use tauri::{Emitter, Manager};

/// Set to true by `shutdown_now` right before it calls `app.exit(0)`.
/// The RunEvent handlers read this to decide whether to allow the
/// exit (our intentional shutdown) or prevent it (user closed the
/// window / hit Cmd+Q — which should hide-to-tray).
pub static INTENTIONAL_QUIT: AtomicBool = AtomicBool::new(false);

pub fn mark_intentional_quit() {
    INTENTIONAL_QUIT.store(true, Ordering::SeqCst);
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let pm = Arc::new(processes::ProcessManager::new());

    let app = tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .manage(pm)
        .setup(|app| {
            // Capture the login-shell PATH now so the first spawn
            // doesn't pay the probe latency. Critical for the packaged
            // app: launched from Finder it only inherits /usr/bin:/bin:
            // … and would otherwise fail to find git/go/docker/ngrok.
            shellpath::warm();
            // Clean up any spawns the previous session left behind —
            // dev HMR reload, force-quit, crash. Runs before the tray
            // so the tray's "running" indicators reflect the post-clean
            // state.
            processes::clean_orphans_from_prior_run(app.handle());
            // Regular activation policy: dock icon, Cmd+Tab entry,
            // standard window management. The tray icon stays as a
            // secondary entry for status-at-a-glance and quick
            // start/stop while the window is behind something.
            tray::build(app.handle())?;
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            settings::get_settings,
            settings::save_settings,
            settings::probe_fleet_repo,
            deps::check_dependencies,
            settings::parse_ngrok_yml,
            settings::read_text_file,
            settings::write_text_file,
            settings::open_path,
            settings::open_url,
            git::git_branch_status,
            git::git_list_branches,
            git::git_fetch,
            git::git_pull,
            git::git_checkout,
            git::git_stash_and_checkout,
            git::git_discard_and_checkout,
            processes::list_processes,
            processes::start_process,
            processes::stop_process,
            processes::restart_process,
            processes::forget_process,
            processes::shutdown_now,
            processes::docker_compose_status,
            processes::docker_compose_down_cmd,
            processes::docker_compose_restart_cmd,
            processes::serve_tcp_check,
            processes::read_log_window,
            processes::clear_log_channel,
            processes::save_log_snapshot,
            processes::logs_dir_path,
            db::db_backups_dir,
            db::db_ensure_backups_dir,
            db::db_list_backups,
            db::db_save_backup_meta,
            db::db_delete_backup,
            db::db_check_backup_name,
            fleetctl::fleetctl_resolve_binary,
            fleetctl::fleetctl_read_context,
            fleetctl::fleetctl_read_config_raw,
            fleetctl::fleetctl_save_config,
            fleetctl::fleetctl_run_capture,
            troubleshoot::troubleshoot_scan_port,
            troubleshoot::troubleshoot_scan_pattern,
            troubleshoot::troubleshoot_kill_pid,
            perf::perf_list_templates,
            gitops::gitops_list_repos,
            gitops::gitops_check_target,
            tray::update_tray,
        ])
        .build(tauri::generate_context!())
        .expect("error while building tauri application");

    app.run(|app_handle, event| match event {
        // X button / Cmd+W: hide the window but keep the app alive.
        // We deliberately do NOT flip the activation policy to
        // Accessory here — that hides the dock icon but breaks the
        // icon image on the way back (NSApp loses the bundle icon
        // reference after a runtime policy switch, falling back to
        // the executable name). Better to keep the dock icon visible
        // while hidden, and let the Reopen handler below catch
        // dock-click to bring the window back. Same pattern as
        // Slack / Discord.
        tauri::RunEvent::WindowEvent {
            event: tauri::WindowEvent::CloseRequested { api, .. },
            ..
        } if !INTENTIONAL_QUIT.load(Ordering::SeqCst) => {
            api.prevent_close();
            if let Some(w) = app_handle.get_webview_window("main") {
                let _ = w.hide();
            }
        }
        // macOS dock-icon click while no windows are visible. Without
        // this handler the click is a no-op once the user has hidden
        // the window via X / Cmd+W.
        #[cfg(target_os = "macos")]
        tauri::RunEvent::Reopen {
            has_visible_windows,
            ..
        } if !has_visible_windows => {
            tray::show_main_window(app_handle);
        }
        // Cmd+Q, app menu > Quit, dock right-click > Quit. Routes to
        // the same quit-confirm flow tray > Quit uses so services get
        // cleaned up before exit.
        tauri::RunEvent::ExitRequested { api, .. }
            if !INTENTIONAL_QUIT.load(Ordering::SeqCst) =>
        {
            api.prevent_exit();
            let _ = app_handle.emit("app:quit-requested", ());
        }
        _ => {}
    });
}
