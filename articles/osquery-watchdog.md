# Osquery watchdog

Osquery will run a watcher process to keep track of any child process and any managed extensions. What follows is a description of what happens during the watcher REPL and under what circumstances the child process and/or managed extensions are terminated.

As a first step, the watcher checks the state of the child worker process, which could be either `Alive` or `Non-existent`. If the process is `Alive`, we make sure the process is within its assigned resource quota, by checking: 

1. That the maximum CPU utilization limit is not exceeded (which is controlled by osquery's `--watchdog_latency_limit` flag).

2. The maximum memory limit is not exceeded (which is controlled by osquery's `--watchdog_memory_limit` flag).
	   
If the child process is within the resource limits, then it is deemed alive and well. Otherwise, we terminate the process by following these steps:
1. We send a `SIGUSR1` to the child process.
2. We send a `SIGTERM` to the child process.
3. After a delay (configured by osquery's `--watchdog_forced_shutdown_delay` flag) we send a `SIGKILL` to the child process.

If the child process is `Non-existent`, either because it didn't exist in the first place or because it was terminated, the watcher will try to spawn a new child process. But first, it will check whether the maximum number of allowed process re-spawns was reached. If it was, then the osquery process shutdowns.

After checking the state of the child worker, we check the state of every managed extension, which could be `Alive` or `Non-existent`.

If the managed extension is `Alive`, the watcher will check both the CPU utilization and memory consumption (the same checks we perform for the child process). If the managed extension is deemed unstable, we terminate the extension by following these steps:
1. We send a `SIGTERM` to the managed extension.
2. After a delay (configured by osquery's `--watchdog_forced_shutdown_delay` flag), we send a `SIGKILL` to the managed extension.

If the managed extension is `Non-existent` (either because it was `Non-existent` in the first place or because it was terminated due to resource contention), the watcher will try to 'launch' the managed extension. But first, it will check the respawn limit. If the respawn limit was reached or if for some reason the extension could be spawned, then the osquery process is shut down.

Lastly, we check the state of the watcher process itself. If it is deemed unhealthy because of resource contention, then the osquery process is shut down.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="juan-fdz-hawa">
<meta name="authorFullName" value="Juan Fernandes">
<meta name="publishedOn" value="2023-07-28">
<meta name="articleTitle" value="Osquery watchdog">
<meta name="description" value="Learn about how osquery process manages child processes and managed extensions in Fleet.">
