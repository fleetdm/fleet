# Vendor questionnaires

## Scoping
| Question | Answer                                                                                                                                                 |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Will Fleet allow us to conduct our own penetration test?   | Yes                                                               |


## Application security
| Question | Answer                                                                                                                                                 |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Does Fleet use any third party code, including open source code in the development of the scoped application(s)? If yes, please explain.   | Yes. All third party code is managed through standard dependency management tools (Go, Yarn, NPM) and audited for vulnerabilities using GitHub vulnerability scanning.                                                                                                                              |

## Data security
| Question | Answer                                                                                                                                                 |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Should the need arise during an active relationship, how can our Data be removed from the Fleet's environment?   | Customer data is primarially stored in RDS, S3, and Cloudwatch logs. Deleting these resources will remove the vast majority of customer data. Fleet can take further steps to remove data on demand, including deleting individual records in monitoring systems if requested.                                                                                                                              |
| Does Fleet support secure deletion (e.g., degaussing/cryptographic wiping) of archived and backed-up data as determined by the tenant? | Since all data is encrypted at rest, Fleet's secure deletion practice is to delete the encryption key. Fleet does not host customer services on-premise, so hardware specific deletion methods (such as degaussing) do not apply. |
| Does Fleet have a Data Loss Prevention (DLP) solution or compensating controls established to mitigate the risk of data leakage? | In addition to data controls enforced by Google Workspace on corporate endpoints, Fleet applies appropiate security controls for data depending on the requirements of the data, including but not limited to minimum access requirements. |

## Encryption and key management
| Question | Answer                                                                                                                                                 |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Does Fleet have a cryptographic key management process (generation, exchange, storage, safeguards, use, vetting, and replacement), that is documented and currently implemented, for all system components? (e.g. database, system, web, etc.)   | All data is encrypted at rest using methods appropiate for the system (ie KMS for AWS based resources). Data going over the internet is encrypted using TLS or other appropiate transport security. |
| Does Fleet allow customers to bring and their own encryption keys? | By default, Fleet does not allow for this, but if absolutely required, Fleet can accommodate this request. |

## Governance and risk management
| Question | Answer                                                                                                                                                 |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Does Fleet have documented information security baselines for every component of the infrastructure (e.g., hypervisors, operating systems, routers, DNS servers, etc.)?  | YWe follow best practices for the given system. For instance, with AWS we utilize AWS best practices for security including GuardDuty, CloudTrail, etc.                                                                |

## Network security
| Question | Answer                                                                                                                                                 |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Does Fleet have the following employed in their production environment? File integrity Monitoring (FIM), Host Intrusion Detection Systems (HIDS), Network Based Indrusion Detection Systems (NIDS), OTHER?   | Fleet utilizes several security monitoring solutions depending on the requirements of the system. For instance, given the highly containerized and serverless environment, FIM would not apply. But, we do use tools such as (but not limited to) AWS GuardDuty, AWS CloudTrail, and VPC Flow Logs to actively monitor the security of our environments.                                                               |

## Privacy
| Question | Answer                                                                                                                                                 |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Is Fleet a processor, controller, or joint controller in its relationship with its customer?  | Fleet is a processor.                                                               |

## Sub-processors
| Question | Answer |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Does Fleet possess an APEC PRP certification issued by a certification body (or Accountability Agent)? If not, is Fleet able to provide any evidence that the PRP requirements are being met as it relates to the Scoped Services provided to its customers? | Fleet has not undergone APEC PRP certification but has undergone an external security audit that included pen testing.  |

<meta name="maintainedBy" value="dherder">
<meta name="title" value="📃 Vendor Questionnaires">
