import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import Nav from "@/components/Nav";
import { TechnicalHelpProvider } from "@/components/TechnicalHelp";
import { ThemeProvider } from "@/components/Theme";
import CommandPalette from "@/components/palette/CommandPalette";
import { PaletteProvider } from "@/components/palette/PaletteContext";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Fleet Atlas",
  description:
    "Interactive maps, scenarios, and self-assessments for learning how Fleet works.",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="en"
      suppressHydrationWarning
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
    >
      <body className="min-h-full bg-zinc-50 text-zinc-900 dark:bg-zinc-900 dark:text-zinc-100">
        <ThemeProvider>
          <PaletteProvider>
            <TechnicalHelpProvider>
              <CommandPalette />
              <div className="flex min-h-screen">
                <Nav />
                <main className="flex-1 overflow-x-hidden">{children}</main>
              </div>
            </TechnicalHelpProvider>
          </PaletteProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
