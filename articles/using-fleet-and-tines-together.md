# Using Fleet and Tines together

Not long ago, I had never heard of Tines. I would have happily created custom python scripts to parse data coming from a REST API and have been done with it. Sure, accomplishing a data ingestion and transformation initiative with python is something that many people still do and, in some cases, still needed. The problem that Tines solves for me is the quick connection and data transformation use case when getting data out of the Fleet REST API. With Tines, I can quickly access all of the endpoints available in Fleet and use a simple lo-code solution to solve problems. Plus, the interface is stellar!

Fleet collects a large amount of data, and in many cases, it is not possible (or feasible) to represent that data in the UI in a way that solves all user needs. In the Fleet UI, the relationship between hosts and software installed on those hosts is illustrated in a couple of ways. 

First, by leveraging the “Software” tab, you can add a filter that will display only the software that has detected vulnerabilities.

![](https://lh4.googleusercontent.com/TL_zTMGVtDdSHVw23yabupYSity_WU15JuoaO5N4rzHdN_xl4vM-LI8BmVx-5Mu4JAOJEQpj6lzSDTaS-dIHnpltptELeOOGtCW089rqyxvG1PT761OF4GARjneMwB4xbLZyFMJXqkqPvEeIlh76Xig)

From the “Hosts” column, you can see the count of hosts in the respective team that have the vulnerable software installed. If you click “View all hosts,” you can get the list of hosts displayed. Additionally, in this view, you can export the list of hosts to take further action toward remediation or compliance reporting. 

![](https://lh5.googleusercontent.com/-LsQzMyiZK3Qb9apskZtWSMMWh6Sm58_WlKpZV_gbrEDb--q77R2g_ZSg746R88qhmNGc_z9AdFdgbJ1E1zaVsNUQGI_khZhxI5_LSymMYEt8l1uNAR9vjKUNNf8gm5dONXWguka-IpUEv1thMgqZYo)

But what if I wanted that list of hosts (in the above example, there is only one host that has the vulnerable software “UTM.app” installed) as well as the CVSS score, probability of exploit, and CVE numbers associated with the list of hosts? Or, just show me a list of hosts with all vulnerable software and the related CVE?

The good news is that Fleet has all of this information, and it is just a matter of transforming the data returned from its REST API to get it. Normally, this is where I would shift into a PyCharm IDE and start building something with requests. Enter Tines, and a bunch of problems just get solved without having to maintain a bunch of code. 

Out of the box, Tines just knows about Fleet. They have a concept called “Templates,” and if you simply search “fleet” in the Templates list, all of the endpoints that Fleet offers just automagically appear. 

![](https://lh4.googleusercontent.com/bFlszLc1GNr02GWuwhDnM5TH8YmtDLOtRL_B9ASpsgPCr70RZIIfyO5nLEotADmlXe_AeoTw4Ce5MozQFc8i6miATkdhQKQE2gRo00TM04aGNTqzgxAOEpO3toHbZ3eAW65f03hd11mEXN-0btBZYGw)

I will use the HTTP Request element to access the “Get Specific Host” information in this example. In order to query all the information about a host, I simply need the Host ID or UUID. The Host ID is accessible by navigating to the host's detail page in Fleet, and the ID will be present in the browser as `https://<fleetserver>/hosts/<host_id>`. Alternatively, you can get the Host UUID from the columns list on the main Hosts page.

![](https://lh6.googleusercontent.com/7K2i3YAFMeGrN7cElyqAhLWtN43Yq1e-VWPir-pKlWnA_Lf5MiW3o0Kq1x1k9xVb1ZNmBAdobxfk9lVDF8qIbeg4fLRWpWXIuCJaF3K1SEOq-vnoSwA9bxNjt_HTsWyXeUWNr7FPp3xudCk1xNC9yu4)

Before we can make a query in Tines, we will need to get an API key and user setup in Fleet. Please refer to our [API documentation regarding Authentication](https://fleetdm.com/docs/using-fleet/rest-api#retrieve-your-api-token). Once you have your API token, head over to Tines and add it to the Credentials section. Out of the box, there will be a text-based credential item that is used in the Tines “fleet” Templates called “fleet\_dm\_API\_key”. In the “Credentials” section of Tines, you will need to create this credential to match what the templates are referencing (fleet\_dm\_API\_key).

![](https://lh5.googleusercontent.com/2H_zv0nUn6Y5aTasmraqJmyNTp7_9JOaXSV9JG92SOtREsINLJGTzI__hEpvaRLWoO9zxUrV9P_RdF_zPIkhG65NoJitenibs2EVY-qIoi1JINcwYaBOmFb1-nPJBWIVwH1ffFux38bvbp4dPKEeSO0)

Enter “Bearer \<your token>” in the “Value” field, where \<your token> is your token from Fleet. The last configuration that needs to be done to get Tines to talk to your Fleet instance is the “fleet\_domain,” which is essentially the domain name of your Fleet instance. To create this resource, navigate to the “Resources” section in Tines and create a new instance called “fleet\_domain” and populate the value with the domain name of your fleet server.

![](https://lh6.googleusercontent.com/-d-yKhCp8-glZ2Nk5BocHleLCDEfcOIBqM_3-ewEPw1fjwzpw4rjIgwxxwA1wVDWCCL24VHNS-eg1XIAGuQrtkhbWiy8HXxhURr2ayiKr9U1rTFEWB1PDQasQF1yEFwbCYu2utisOUuvIpPeOgPqA0E)

Now that you have Tines configured to pre-populate the Fleet API token and server name, let us build on our example above by adding an “Event Transform” so that we can take the Event Response of the “Get Specific Host” query and remove any software from the list where vulnerabilities do not exist. In order to accomplish this, add the following code to the Payload Builder:

```
WHERE(get_specific_host.body.host.software,'vulnerabilities')
```

![](https://lh5.googleusercontent.com/DDFqEq7--nxVpZUbuQg573O5JeCA5f1ERqVcM9IzBhpxYE-pfay64EcO05GaHPFsP543Vg1EOy259dXYTkDaohOyjS2LhX4QeIcMSB5nAn8wxPjjUmWjfnQicowLNJfG-KYDCp_ZbF8TawylknwkfRA)

Next, I only want to generate alerts or send emails to users where vulnerability CVSS is greater than 9.0 (Critical). To do this, I can use the handy JSONPATH function to traverse the JSON structure and add a filter of sorts.

```
JSONPATH(vulnerabilites_not_null, "*.vulnerabilities[?(@.cvss_score>9.0)]")
```

![](https://lh6.googleusercontent.com/VneLd3XEZn3KG_A-XsiqclYGWblTpOx4CN6l_SvmptZlBf2EHmFPE7fWQGtD7mwwXpnIKe37akHFRbqNuyALk3cTc-JduOzoE6PBcyOaFVvsN4tg3hSPoJzkoAKGfrwl2-_7XHrvsr82ptLgS7MnGmU)

Lastly, I’m going to use the response from the previous query to build my “Send Email” function. 

In the Tines Editor, the formatting would look something like this:

```
{
  "recipients": [
    "you@example.com"
  ],
  "reply_to": "you@example.com",
  "sender_name": "Dave Herder",
  "subject": "Send Email",
  "body": "<b>Host:</b> <<get_specific_host.body.host.hostname>><br>\n\n<b>Host ID:</b> <<get_specific_host.body.host.id>><br>\n\n<b>Software with Critical Vulnerabilities: </b><br>\n<%for software in vulnerabilites_not_null%><<software.name>><br> <%for vulnerability in filter_list_of_software_with_cvss_9_0%><<vulnerability.cve>> (CISA known exploit?    <<vulnerability.cisa_known_exploit>>)\n<br><%endfor%><br>\n<%endfor%>"
}
```

![](https://lh5.googleusercontent.com/u5r1am-5XinXf0cMyIPz86HxV1Ep7f_UXWQrilYfJ859o5LEisd6gRbsGmocETaroMy8uHCSC14pWCgHpiY5Zckv2arOHH0mNbyc1fN0WZ3wjtsPmM4Y9wRlHZB11ch_WxokwVWJWzxy1luWWgep_WA)

The final email with the above definition looks like this:

![](https://lh3.googleusercontent.com/ZNV-ivzt1IvvGCFyk6iGto_QpCSsL7S-VCakwXzzKc-9ZCoyRooIr3kEkukWDCWD7kqOs0ZlLFuvsCU5ar4jv_JmnF92EKT51mICsqzyNdk_eRYN9L651zQiIMC92TsO5Q4AvY_VC20j79EnCYPDPcM)

The Fleet API is very flexible, but with the addition of Tines, the options for data transformation are endless. In the above example, we easily connected to the Fleet API and transformed the data response with a single Tines Transform function, and allowed the end user to receive a customized report of vulnerable software on an individual host.
