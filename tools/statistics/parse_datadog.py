import pandas as pd
import json
import altair as alt
import webbrowser as browser

DATADOG_CSV = "https://data.heroku.com/dataclips/tzvzqkwgaglcibrogcuzsnkhssnx.csv"
simplify_macos = False
top_versions = 10

pd.set_option("display.max_rows", None)
pd.set_option("display.max_columns", None)

# Read the CSV file into a DataFrame
df = pd.read_csv(DATADOG_CSV)

# Extract the "hostsEnrolledByOperatingSystem" column as a series
hosts_enrolled_series = df["hostsEnrolledByOperatingSystem"]

print(f"Keys: {df.keys()}")

print(f"Total Hosts   : {df['numHostsEnrolled'].sum()}")
print(f"Total Teams   : {df['numTeams'].sum()}")
print(f"Total Users   : {df['numUsers'].sum()}")
print(f"Total Queries : {df['numQueries'].sum()}")
print(f"Total Policies: {df['numPolicies'].sum()}")
print(f"Total Labels  : {df['numLabels'].sum()}")

# Print the first 5 elements of the series
print(hosts_enrolled_series.head())


# Define the function to extract version and count from the JSON string
def extract_version_and_count(json_str):
    try:
        data = json.loads(json_str)
        version_counts = {}
        for platform, versions in data.items():
            for version in versions:
                # print(platform, version)
                version_str = version.get("version")
                num_enrolled = version.get(
                    "numEnrolled", 0
                )  # Handle missing 'numEnrolled'
                if simplify_macos:
                    if version_str and version_str.lower().startswith("macos"):
                        if version_str.lower().startswith("macos 10"):
                            version_str = "macOS 10.xx"
                        if version_str.lower().startswith("macos 11"):
                            version_str = "macOS 11.xx"
                        if version_str.lower().startswith("macos 12"):
                            version_str = "macOS 12.xx"
                        if version_str.lower().startswith("macos 13"):
                            version_str = "macOS 13.xx"
                        if version_str.lower().startswith("macos 14"):
                            version_str = "macOS 14.xx"
                        if version_str.lower().startswith("macos 15"):
                            version_str = "macOS 15.xx"
                    if version_str and version_str.lower().startswith("macos"):
                        version_counts[version_str] = (
                            version_counts.get(version_str, 0) + num_enrolled
                        )
                else:
                    if version_str:
                        # if version_str not in version_counts:
                        # print(platform, version_str)
                        version_counts[version_str] = (
                            version_counts.get(version_str, 0) + num_enrolled
                        )
        return version_counts
    except json.JSONDecodeError:
        return {}


# Apply the function to each element of the series and store the results in a list
version_counts_list = [
    extract_version_and_count(item) for item in hosts_enrolled_series
]

# print(json.dumps(version_counts_list[0], indent=2))

# Initialize an empty dictionary to store the total count for each version
total_version_counts = {}

# Iterate through the list of dictionaries and aggregate the counts
for version_counts in version_counts_list:
    for version, count in version_counts.items():
        total_version_counts[version] = total_version_counts.get(version, 0) + count

# if top_versions is not 0, sort the dictionary by value and keep only the top versions
if top_versions != 0:
    total_version_counts = dict(
        sorted(total_version_counts.items(), key=lambda item: item[1], reverse=True)[
            :top_versions
        ]
    )


# Create a DataFrame from the main dictionary
version_counts_df = pd.DataFrame.from_dict(
    total_version_counts, orient="index", columns=["numEnrolled"]
)

# pretty print total_version_counts
# print(json.dumps(total_version_counts, indent=2))

# Sort the DataFrame by the "numEnrolled" column in descending order
version_counts_df = version_counts_df.sort_values(by="numEnrolled", ascending=False)

# Plot a bar chart with "version" on the x-axis and "numEnrolled" on the y-axis
chart = (
    alt.Chart(version_counts_df.reset_index())
    .mark_bar()
    .encode(x="index", y="numEnrolled", tooltip=["index", "numEnrolled"])
    .properties(title="Total Enrolled Hosts by Version")
    .interactive()
)

# Save the chart
chart.save("total_enrolled_hosts_by_version_bar_chart.html")

# open the chart in the browser
browser.open_new_tab("total_enrolled_hosts_by_version_bar_chart.html")
