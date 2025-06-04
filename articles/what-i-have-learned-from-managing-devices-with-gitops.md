In today's rapidly evolving technology landscape, managing device configurations at scale presents significant challenges for IT operations and Client Platform Engineering teams. GitOps—applying git-based workflows and DevOps principles to configuration management—offers a powerful solution to these challenges. By treating infrastructure and device configurations as code stored in a git repository, organizations can achieve levels of consistency, reliability, and automation that are unseen in most modern SaaS applications. This article explores the key lessons I've learned implementing GitOps for device management, highlighting how this approach transforms traditional operations through declarative configurations, automated workflows, enhanced security, and fundamental cultural shifts. Whether you're managing a small fleet of devices or thousands of endpoints across diverse platforms, GitOps principles can revolutionize your approach to configuration management.

## **Power of Declarative configuration**

One of the core tenets of GitOps is declarative configuration, where the desired state of the system is explicitly defined in code. In the case of Fleet, these are easy-to-understand YAML files that clearly represent the intended configuration. This approach ensures consistency across all managed devices and provides several key advantages:

* **Single source of truth**: All configuration lives in a version-controlled repository, eliminating uncertainty about which configuration is currently deployed.  
* **Configuration as code**: Treating configuration as code allows teams to leverage software development best practices for infrastructure management.  
* **Reduced configuration drift**: Instead of manually configuring each device, GitOps enforces the defined desired state, automatically correcting deviations when they occur.  
* **Improved scalability**: As your device fleet grows, the same configuration can be consistently applied to hundreds or thousands of devices without proportional management overhead.

## **Automation simplifies deployment and rollback**

By automating device configuration and updates through GitOps, deployments become predictable and repeatable. This leads to significantly enhanced operational efficiency for several reasons:

* **Continuous delivery pipeline**: Changes to configuration automatically flow through a standardized pipeline, ensuring consistent validation and deployment processes.  
* **Reduced manual intervention**: Automation eliminates error-prone manual steps, freeing team members to focus on higher-value activities.  
* **Self-healing systems**: GitOps continuously reconciles between the actual and desired states, automatically correcting any unauthorized changes.  
* **Simplified rollbacks**: In case an unexpected issue is introduced, rolling back to a previously known good working state is as simple as reverting a git commit, reducing downtime and risk.  

## **Security and compliance benefits**

GitOps inherently improves security by enforcing strict access control and audit trails. This approach provides several security advantages:

* **Comprehensive audit trail**: With all changes recorded in git, there is a clear history of who made what changes and when, satisfying audit requirements and simplifying troubleshooting.  
* **Separation of duties**: Using policies like requiring different individuals for change submission, review, approval, and production deployment creates robust security guardrails.  
* **Automated validation checks**: Pre-deployment checks can verify that changes meet security, compliance, and organizational standards before reaching production.  
* **Reduced attack surface**: By limiting direct access to systems and enforcing changes through the GitOps pipeline, you significantly reduce potential attack vectors.  
* **Secrets management**: Integration with secrets management tools ensures sensitive information is handled securely throughout the deployment process.

## **Speak one language**

GitOps establishes a common language and framework for cross-functional teams to collaborate effectively:

* **Unified workflow**: Both development and operations teams work through the same GitOps process, breaking down traditional silos.  
* **Consistent terminology**: Standardized terms and concepts improve communication across different specialties and reduce misunderstandings.  
* **Visible changes**: Pull requests make proposed changes visible to all stakeholders, enabling broader input before implementation.  
* **Knowledge transfer**: The repository becomes a living knowledge base, documenting not just what configurations exist but why they were implemented.  
* **Accessible history**: New team members can review the evolution of configurations to understand the reasoning behind current implementations.

## **Software management**

GitOps principles extend naturally to software deployment and management across device fleets:

* **Reproducible environments**: The combination of configuration and software definitions creates fully reproducible environments.  
* **Progressive delivery**: Implementing canary deployments or blue-green strategies for software updates reduces risk.  
* **Dependency management**: Tracking software dependencies in git helps identify potential conflicts or security vulnerabilities.  
* **Automated testing**: Integration with CI/CD pipelines enables automated testing of software packages before deployment to devices.

# **GitOps culture shift**

Successful GitOps adoption requires a fundamental cultural shift within the team. While the technical aspects are important, the organizational and mindset changes are equally crucial for realizing the full benefits of GitOps in device management.

### **Collaborative development through pull requests**

Implementing a pull request workflow encourages team collaboration and knowledge sharing. This process creates natural checkpoints where team members can discuss changes, offer improvements, and ensure alignment with organizational goals before deployment to production environments. It also establishes a clear paper trail of who requested what changes and why, creating accountability and transparency across the organization.

### **Code review as a quality gate**

Enforcing mandatory code reviews serves as a critical quality gate that prevents configuration drift and ensures consistency. By establishing clear review criteria focused on security, compliance, and operational best practices, teams can catch potential issues early in the development lifecycle. Code reviews also serve as excellent mentoring opportunities, allowing experienced engineers to share knowledge with newer team members.

### **Automation mindset**

GitOps thrives when teams embrace an "automate everything" philosophy. This involves identifying manual processes that can be automated and continuously refining automation workflows. Teams should measure and celebrate reductions in manual interventions as key performance indicators of successful GitOps implementation.

### **Documentation as code**

Treating documentation as code within the same repository as configuration ensures that documentation stays current with the actual system state. This approach facilitates onboarding new team members and provides clear context for future changes. When documentation lives alongside code, it's more likely to be maintained as systems evolve.

### **Continuous learning culture**

Successful GitOps teams foster a continuous learning environment where failure is viewed as an opportunity for improvement rather than blame. Post-incident reviews should focus on improving processes and automation rather than finding individual fault, encouraging team members to report issues openly and collaborate on solutions.

### **Cross-functional ownership**

Breaking down silos between development, operations, and security teams is essential for GitOps success. Shared ownership of the GitOps workflow ensures that all perspectives are represented in the final configurations, leading to more robust and secure systems.

## **GitOps mode in Fleet**

The [introduction of GitOps mode in Fleet 4.65.0,](https://fleetdm.com/releases/fleet-4-65-0) represents a significant advancement in Fleet's enterprise readiness and GitOps capabilities. By allowing administrators to place the Fleet UI in read-only mode, the feature creates a clear separation between operational visibility and configuration management, ensuring all changes follow proper version control protocols through their git repository. This is particularly valuable in enterprise environments where configuration drift can lead to security vulnerabilities and compliance issues. This feature also solves the common problem of conflicting changes that occur when some team members modify settings through the UI while others manage configurations through configuration-as-code pipelines. By directing users to the git repository for changes, GitOps mode fosters better collaboration, maintains a single source of truth for configurations, and strengthens audit trails \- all critical requirements for mature security operations. This alignment with modern GitOps and change management practices demonstrates Fleet's evolution from a simple device management tool to an enterprise-grade platform that can integrate seamlessly with sophisticated IT governance frameworks.

## **Final thoughts**

Managing devices with GitOps has transformed the way I approach configuration management and deployment automation. The benefits of consistency, automation, security, and observability make GitOps a compelling methodology for modern device management. By embracing declarative configurations, automating deployments, enhancing security, simplifying platform-specific challenges, and fostering a collaborative culture, organizations can achieve significant operational improvements.

While implementing GitOps requires an initial investment in tooling and cultural change, the long-term advantages far outweigh these costs. The resulting system provides greater reliability, faster recovery from issues, improved security posture, and enhanced development velocity—all critical factors for organizations managing device fleets at scale.

The journey to GitOps adoption may present challenges, but with thoughtful implementation and a commitment to continuous improvement, it can become an invaluable asset for device management and a key differentiator in how your business operates.

---

## **Want to learn more?** 

Reach out directly to me or [the team at Fleet today](https://fleetdm.com/contact)\! 

Check out the [livestream replay for "GitOps: Infrastructure-as-code for managing devices at scale"](https://www.linkedin.com/events/gitops-infrastructure-as-codefo7289751303876952065/comments/).

[Sign up for free in-person GitOps training](https://www.eventbrite.com/cc/gitops-for-device-management-4104123) in your city. 


<meta name="articleTitle" value="What I have learned from managing devices with GitOps">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-03-28">
<meta name="description" value="GitOps changes more than just technology.">
