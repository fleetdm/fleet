import ArchitectureTabs from "@/components/arch/Tabs";

export default function ArchitectureLayout({
  children,
}: LayoutProps<"/architecture">) {
  return (
    <div className="flex min-h-screen flex-col bg-zinc-50 dark:bg-zinc-900">
      <header className="border-b border-zinc-200 bg-white px-6 py-4 dark:border-zinc-800 dark:bg-zinc-950">
        <h1 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
          Architecture map
        </h1>
        <p className="mt-1 text-xs text-zinc-500">
          How Fleet is built, viewed as structural maps or as quizzes drawn from
          each map&apos;s nodes.
        </p>
      </header>
      <ArchitectureTabs />
      <div className="flex-1">{children}</div>
    </div>
  );
}
