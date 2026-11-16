// Keeps the GUI's settings in step with the Go store. The store can change a
// reserved slot's identity during a save, so every save is followed by a
// reload. Each request takes a generation; only the latest result is applied,
// so a slow earlier reload can't overwrite a newer one.
export function createSettingsSync<T>(load: () => Promise<T>) {
  let generation = 0;
  let saving = 0;
  let apply: ((s: T) => void) | null = null;

  const reload = async (): Promise<void> => {
    const mine = ++generation;
    try {
      const s = await load();
      if (mine === generation && saving === 0) apply?.(s);
    } catch (e) {
      console.error("settings reload failed", e);
    }
  };

  return {
    setApply(fn: ((s: T) => void) | null) {
      apply = fn;
    },
    reload,
    // Requests the reload whether the save succeeded or failed. The request
    // is deferred a macrotask so even an immediately-resolving load applies
    // after the caller's own onChange(next).
    // Reloads resolving while any save is in flight are dropped, so a fast
    // typist's later keystroke isn't overwritten by an earlier save's reload.
    async afterSave<R>(save: () => Promise<R>): Promise<R> {
      saving++;
      generation++;
      try {
        return await save();
      } finally {
        saving--;
        setTimeout(() => void reload(), 0);
      }
    },
  };
}
