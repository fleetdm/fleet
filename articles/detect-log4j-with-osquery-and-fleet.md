# Detect Log4j with osquery (and Fleet)

![Detect Log4j with osquery (and Fleet)](../website/assets/images/articles/detect-log4j-with-osquery-and-fleet-1600x900@2x.jpg)

[Log4j](https://logging.apache.org/log4j/2.x/) is a widely used Java-based logging library that has been under active development since 1999 by The Apache Software Foundation. Security researchers have found a zero-day vulnerability [CVE-2021–44228](https://nvd.nist.gov/vuln/detail/CVE-2021-44228) that is actively being exploited in the wild to take control of an affected computer remotely.

In response, the ASF has released a patch and recommends an immediate upgrade to fix the impacted library.

Since Log4j is an embedded library used by many applications in your server and endpoint environments: how do you know your exposure? Here we describe a query you can run using Fleet to get granular and real-time visibility into Log4j installs across your infrastructure.

<div purpose="embedded-content">
	<iframe src="https://www.youtube.com/embed/pRE_QT5zr5s" allowfullscreen></iframe>
</div>

## The Query (tl;dr)

The Fleet team developed this osquery SQL to detect running processes with Log4J loaded on Linux and macOS systems. Run it as a live query via [Fleet](https://fleetdm.com/) (or any other osquery manager) to quickly detect potential targets within your infrastructure. Thank you to Tim Brown for [creating these YARA queries](https://github.com/timb-machine/log4j).

This query can also be found in Fleet’s osquery [standard query library](https://fleetdm.com/queries/detect-active-processes-with-log-4-j-running).

```
WITH target_jars AS (
  SELECT DISTINCT path
  FROM (
      WITH split(word, str) AS(
        SELECT '', cmdline || ' '
        FROM processes
        UNION ALL
        SELECT substr(str, 0, instr(str, ' ')), substr(str, instr(str, ' ') + 1)
        FROM split
        WHERE str != '')
      SELECT word AS path
      FROM split
      WHERE word LIKE '%.jar'
    UNION ALL
      SELECT path
      FROM process_open_files
      WHERE path LIKE '%.jar'
  )
)
SELECT path, matches
FROM yara
WHERE path IN (SELECT path FROM target_jars)
  AND count > 0
  AND sigrule IN (
    'rule log4jJndiLookup {
      strings:
        $jndilookup = "JndiLookup"
      condition:
        $jndilookup
    }',
    'rule log4jJavaClass {
      strings:
        $javaclass = "org/apache/logging/log4j"
      condition:
        $javaclass
    }'
  );
```

>Note: This query is resource intensive and has caused problems on systems with limited swap space. Test on some systems before running this widely.

## How it works

The query essentially works in two parts:

1. Find loaded Java JAR files on the system.
2. Use YARA scanning to detect Log4J utilization in those files.

### Find JARs

JARs are found via two mechanisms on the host.

This complex-looking syntax actually just splits the command line arguments for each running process on the system, filtering for any arguments ending in .jar:

```
WITH split(word, str) AS(
  SELECT '', cmdline || ' '
  FROM processes
  UNION ALL
  SELECT substr(str, 0, instr(str, ' ')), substr(str, instr(str, ' ') + 1)
  FROM split
  WHERE str != '')
  SELECT word AS path
  FROM split
  WHERE word LIKE '%.jar'
```

These results are combined (using UNION ALL) with the list of open files for each process on the system, filtering again for arguments ending in .jar:

```
SELECT path
FROM process_open_files
WHERE path LIKE '%.jar'
```

### Scan for Log4J

Once the query rounds up all the JARs, we use YARA to scan for evidence of Log4J in those JARs.

A trimmed-down set of the YARA rules from [Tim Brown’s repository](https://github.com/timb-machine/log4j) are applied:

```
SELECT path, matches
FROM yara
WHERE path IN (SELECT path FROM target_jars)
  AND count > 0
  AND sigrule IN (
    'rule log4jJndiLookup {
      strings:
        $jndilookup = "JndiLookup"
      condition:
        $jndilookup
    }',
    'rule log4jJavaClass {
      strings:
        $javaclass = "org/apache/logging/log4j"
      condition:
        $javaclass
    }'
  );
```

<meta name="category" value="security">
<meta name="authorFullName" value="Zach Wasserman">
<meta name="authorGitHubUsername" value="zwass">
<meta name="publishedOn" value="2021-12-15">
<meta name="articleTitle" value="Detect Log4j with osquery (and Fleet)">
<meta name="articleImageUrl" value="../website/assets/images/articles/detect-log4j-with-osquery-and-fleet-1600x900@2x.jpg">
