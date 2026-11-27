import GlossaryTabs from "@/components/glossary/Tabs";
import { TERMS } from "@/components/glossary/terms";

export default function GlossaryLayout({
  children,
}: LayoutProps<"/glossary">) {
  return (
    <div className="mx-auto max-w-5xl px-10 py-10">
      <header className="mb-6 border-b border-zinc-200 pb-6 dark:border-zinc-800">
        <h1 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">
          Glossary
        </h1>
        <p className="mt-2 text-zinc-600 dark:text-zinc-400">
          Fleet terms, protocol acronyms, Go idioms. {TERMS.length} entries.
        </p>
      </header>
      <GlossaryTabs />
      {children}
    </div>
  );
}
