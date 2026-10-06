# No macro prompt, no warning: a malicious spreadsheet runs Java code in LibreOffice before 26.2.5

*A crafted spreadsheet can run attacker code the moment it opens, without the macro warning most users have learned to trust. Here's how to find the builds that still allow it.*

## Key takeaways

- **The warning users rely on never appears.** The attack uses features that each work as designed, so there is no macro prompt to tell a recipient the file is dangerous.
- **LibreOffice has a fix, OpenOffice doesn't.** LibreOffice fixed CVE-2026-63277 in 26.2.5 and 26.8.0, while Apache OpenOffice 4.1.16 and earlier remain exposed to CVE-2026-59265 until 4.1.17 ships.
- **Java is the condition.** The chain only works when Java support is enabled, so the version alone doesn't say whether a given host is at risk.
- **A public proof of concept raises the clock.** Researchers have published a working demonstration, even though no active exploitation has been reported yet.
- **Inventory beats waiting for someone to open the wrong file.** A software inventory tells you which hosts run which build today, so the patch push has a target list.

<a purpose="cta-button" href="https://fleetdm.com/software-catalog">See software inventory in Fleet</a>

LibreOffice and Apache OpenOffice both let a Calc spreadsheet define a database range, a block of cells that pulls data from an outside source and refreshes itself. Researchers Rick de Jager and Thomas Rinsma, working independently, found that this feature can be chained into code execution when the file is opened.

The usual advice for untrusted office files is to watch for a macro warning. This bug removes that signal. So the question shifts from "will the user notice?" to "which of our hosts could be affected at all?"

## How the chain works without a macro

The spreadsheet's database range points at an external ODB database file by URL. That file references a Java database driver (JDBC), and the driver is loaded and run without the prompt a macro would trigger. In the published demonstration, opening the file launched Calculator, but the same path can run any Java code the attacker chooses.

Each step is a documented feature. That's why a user who has learned "no macro warning means safe" has nothing to react to.

## Which versions are affected

| Product | CVE | Status |
|---|---|---|
| LibreOffice before 26.2.5 and before 26.8.0 | CVE-2026-63277 | Fixed in updates released October 5, 2026 |
| Apache OpenOffice 4.1.16 and earlier | CVE-2026-59265 | No fix yet; 4.1.17 is in testing |

Until 4.1.17 ships, OpenOffice users can turn off Java in the program's settings or avoid opening spreadsheets they don't trust. Either way, you need to know where OpenOffice is installed first.

## Why Java settings matter for prioritizing

The attack needs Java support enabled. A host with an affected build and Java disabled is in a different position than one with both, so a version list is a starting point rather than a verdict. Teams that don't use office suites with Java extensions can treat the setting as part of the remediation record.

## Building the target list

Fleet's agent collects installed software and versions from every host it manages, across macOS, Windows, and Linux. Filtering that inventory for LibreOffice and OpenOffice gives you the list of hosts to update, and re-checking after the patch window shows who still carries a vulnerable build. Because Fleet's reports can also read files and settings on a host, you can write one to check the Java setting too, though you should test that report against your own configuration first.

## The signal to replace

Macro warnings were always a weak control, and this bug is a reminder of why. A reliable answer to "who is still running the vulnerable build?" doesn't depend on any user noticing anything.

## See it live

- **Get a demo** to see software inventory across your own hosts: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Explore the software catalog** Fleet builds from every host: [fleetdm.com/software-catalog](https://fleetdm.com/software-catalog)

<meta name="articleTitle" value="No macro prompt, no warning: a malicious spreadsheet runs Java code in LibreOffice before 26.2.5">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-10-06">
<meta name="description" value="A crafted spreadsheet can run code in LibreOffice and OpenOffice with no macro warning. See which builds are exposed and how to find them.">
