# Banking platform guarantees script execution and audit-ready compliance

A banking-as-a-service platform facilitating digital transactions in emerging markets needed to move away from failed legacy tools to a platform with guaranteed script execution.

## At a glance

- **Endpoints:** ~287 (evenly split Mac and Windows).  
- **Primary requirement:** guaranteed remote script execution and patching automation.  
- **Key integrations:** Datadog and Amazon Workspaces.  
- **Previous solution:** Workspace ONE.  

## The Challenge

They experienced "critical failures" with Workspace ONE, including unreliable macOS updates, very limited visibility across standard endpoints and servers,  and inability to monitor remote script output. They also could not manage Amazon Workspaces (VMs), which was a major blind spot.

## The solution

They switched to Fleet for its modern, API-driven workflows. The transparency of the open-source model was essential for compliance tracking in the highly regulated financial sector.

## The results

- **Proving compliance:** transparency improved their ability to prove the state of their fleet with certainty during financial audits.  
- **Customized remediation:** they now use SQL-based osquery queries to automate the "fix" when a device falls out of compliance.  
- **Unified vitals:** direct streaming to Datadog centralizes device vitals within the same dashboards used for their banking infrastructure.

## About Fleet

Fleet is the open-source endpoint management platform that gives you total control, unlike the proprietary 'black boxes' of legacy vendors. Our open device management provides full visibility into our code and roadmap, plus a true choice of deployment—on-prem or cloud—with 100% feature parity. Our API-first approach empowers technical teams to automate with GitOps, scale confidently, and get the real-time data needed to secure their entire macOS, iOS, Windows, and Linux fleets.  

<meta name="articleTitle" value="Banking platform guarantees script execution and audit-ready compliance">
<meta name="authorFullName" value="Irena Reedy">
<meta name="authorGitHubUsername" value="irenareedy">
<meta name="category" value="case study">
<meta name="publishedOn" value="2026-02-22">
<meta name="description" value="A banking platform replaced legacy tools with Fleet to guarantee script execution, unify Mac and Windows, and prove compliance in audits.">
