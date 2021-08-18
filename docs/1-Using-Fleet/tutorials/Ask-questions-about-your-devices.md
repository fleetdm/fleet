# Ask questions about your devices

> This tutorial assumes that you have a preview environment of Fleet up and running. Check out [the "Try Fleet" instructions](../../../README.md#try-fleet) on how to start a preview environment of Fleet.

In this tutorial, we'll cover the following Fleet concepts:

- Where to see your devices in Fleet
- How to ask questions about your devices by running queries

### Where to see your devices in Fleet

Once you log into Fleet, you're presented with the **Hosts** page. In Fleet, devices are refered to as "hosts." 

On this page you'll see 7 hosts by default. These hosts are simulated Linux devices, and like the Fleet preview environment, they're running locally on your computer in Docker. For this tutorial we'll be asking questions about these devices by running some queries against them.


### How to ask questions about your devices by running queries

Let's ask the following questions about the simulated Linux hosts connected to your Fleet:

1. What version of OpenSSL is installed on each device, if any?

2. Do these devices have a high severity vulnerable version of OpenSSL installed?

These questions can easily be answered with Fleet, by running the following query: "Detect Linux hosts with high severity vulnerable versions of OpenSSL." 

On the **Queries** page, enter the query name, "Detect Linux hosts with high severity vulnerable versions of OpenSSL," in the search bar, and select it from the table to navigate to the **Edit or run query** page.


<img src="https://user-images.githubusercontent.com/78363703/128487468-7961c509-d0ba-48be-a0e8-54bfb4c371d5.png" alt="Fleet query search" width="600"/>


On the **Edit or run query** page, open the "Select targets" dropdown, and press the purple "+" icon to the right of "All hosts." This means we'll be attempting to run this query against all hosts connected to your Fleet. 


<img src="https://user-images.githubusercontent.com/78363703/128487638-7d779d89-f3fa-42dd-903f-070dc9347a9b.png" alt="Fleet select targets" width="600"/>


Now hit the "Run" button to run the query, and you're done. The query may take several seconds to complete because Fleet has to wait for the osquery agents to respond with results.

> Fleet's query response time is inherently variable because of osquery's heartbeat response time. This helps prevent performance issues on hosts.

When the query has finished, you should see 4 columns and several rows in the "Results" table:


<img src="https://user-images.githubusercontent.com/78363703/128488112-56c762da-5029-42d1-8f5d-e74f22aa39cd.png" alt="Fleet query results" width="600"/>


- The "hostname" column answers: which device responded for a given row of results? 

- The "name" column answers: what is the name of the installed software item? The query we just ran asked for all software items that contain "openssl" in their name, so each row in this column should contain "openssl."

- The "source" column answers: which osquery table is the result coming from? For more information on the table's available in osquery, check out the [osquery schema documentation](https://osquery.io/schema).

- The "version" column answers: which version of the software item was detected on this device?

The "Results" table presented in Fleet answers our first question of interest which was "What version of OpenSSL is installed on each device, if any?"

Now you have the results from your query, you can compare the results from the "version" column to the table below, which includes the high severity vulnerabilities reported by [OpenSSL](https://www.openssl.org/news/vulnerabilities.html).


| OpenSSL version range                                                  | Vulnerability (CVE)                                                                           |
| --------------------------------------------------------- | ----------------------------------------------------------------------------- |
| 1.1.1h-1.1.1j                                             | [CVE-2021-3450](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2021-3450) |
| 1.1.1-1.1.1j                                              | [CVE-2021-3449](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2021-3449) |
| 1.1.1-1.1.1h and 1.0.2-1.0.2w                             | [CVE-2020-1971](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2020-1971) |
| 1.1.1d-1.1.1f                                             | [CVE-2020-1967](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2020-1967) |
| 1.1.1-1.1.1d and 1.0.2-1.0.2t                             | [CVE-2019-1551](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2019-1551) |
| 1.1.1-1.1.1c                                              | [CVE-2019-1549](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2019-1549) |
| 1.1.0-1.1.0d                                              | [CVE-2017-3733](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2017-3733) |
| 1.1.0-1.1.0b                                              | [CVE-2016-7054](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2016-7054) |
| 1.1.0 and 1.0.2-1.0.2h and 1.0.1-1.0.1t                   | [CVE-2016-6304](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2016-6304) |
| 1.0.2-1.0.2b and 1.0.1-1.0.1n                             | [CVE-2016-2108](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2016-2108) |
| 1.0.2-1.0.2f and 1.0.1-1.0.1r                             | [CVE-2016-0800](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2016-0800) |
| 1.0.2 and 1.0.1-1.0.1l and 1.0.0-1.0.0q and 0.9.8-0.9.8ze | [CVE-2016-0703](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2016-0703) |
| 1.0.2-1.0.2e                                              | [CVE-2016-0701](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2016-0701) |
| 1.0.2b-1.0.2c and 1.0.1n-1.0.1o                           | [CVE-2015-1793](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2015-1793) |
| 1.0.2                                                     | [CVE-2015-0291](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2015-0291) |
| 1.0.1-1.0.1i                                              | [CVE-2014-3513](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2014-3513) |
| 1.0.1-1.0.1h                                              | [CVE-2014-3511](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2014-3511) |
| 1.0.1-1.0.1h                                              | [CVE-2014-3511](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2014-3511) |

Do any of the simulated, Linux hosts have a high severity vulnerable version of OpenSSL installed? If the answer is yes, don't worry. The devices are running in a simulated Docker environment and do not provide any additional vectors for performing malicious actions against your device.
