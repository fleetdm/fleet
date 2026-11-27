"use client";

import { createContext, useCallback, useContext, useState } from "react";

type Ctx = {
  open: boolean;
  setOpen: (v: boolean) => void;
  toggle: () => void;
};

const PaletteCtx = createContext<Ctx | null>(null);

export function PaletteProvider({ children }: { children: React.ReactNode }) {
  const [open, setOpen] = useState(false);
  const toggle = useCallback(() => setOpen((v) => !v), []);
  return (
    <PaletteCtx.Provider value={{ open, setOpen, toggle }}>
      {children}
    </PaletteCtx.Provider>
  );
}

export function usePalette() {
  const ctx = useContext(PaletteCtx);
  if (!ctx) throw new Error("usePalette outside PaletteProvider");
  return ctx;
}
