
# Fleet Team Builder

Using a list of teams in a file as input, adds the listed teams to Fleet and generates installer processes. 

For each team, an enroll secret will be created, the team added to Fleet using the team yaml template, and `.msi`,`.deb`, `.pkg` and `.rpm` installer packages will be created. 

## Requirements

fleetctl installed and logged in

## Flags 

Required flags:

- -s: The source file containing teams to be added. 
- -u: The url of the Fleet server.

Optional flags:

- -h: header - Default: false - Indicates that provided csv file has a header defining columns on the first line.
- -c: columns - list the columns included in the CSV file. For use when header is not included.
- -n: name_column - Default: "Name" - The column that contains the team name.
- -p: packages - Default: "all" - The types of installer packages to create for each team.
- -f: flags - Additional flags to apply to `fleetctl package`.
- -o: output - Default: Current location - Directory in which to place the generated packages.
- -x: dry_run - Test prossesing the file, creating the team in Fleet, and generating packages without applying any changes to the server.

## Usage

### Basic

Create a file including a list of teams, one per line:

```
Workstation
Canary
Servers

```
Run the script and pass the Fleet Server URL and source file as arguments:

```console
$ ./build_teams.sh -s example.txt -u fleet.org.com
```

### Advanced 

Describe the contents of an existing .csv file to import.  

#### CSV With a header line
 
Indicate that a header is present with `-h true` and indicate which column contains the team name using `-n`:

```console
$ ./build_teams.sh -s test.csv -u fleet.org.com -h true -n Name
```

The first line of the source file will be parsed to pull out the columns present and the team name will be extracted from the indicated column.

#### CSV Without a header line

Define the columns present in the file using `-c` and indicate the column that contains the team name with `-n`:

```console
$ ./build_teams.sh -s test.csv -u fleet.org.com -c "Name Code" -n Name    
```

## Team configuration

By default, the template stored in `team_config.yml` will be applied. This file only creates the team and adds a randomly generated enroll secret. The team will inherit global agent options. You can modify the file to include additional options.

If you modify the agent options, please note the placeholders for name and secret, these are necessary for the appropriate vales generated in the script to be applied.

```yml   
    name: ${name}
    secrets: 
      - secret: ${secret}
```

> If you modify the configuration file, please test the script using `-x` before applying changes.

## Testing

To test team creation and package generation without applying the changes to Fleet, include the `-x` flag. This will add the `--dry_run` flag to `fleetctl apply`. All actions will be taken, but the generated team configuration YAML will be validated without creating the new team. 

## TODO:

- Add logic to check if a team with that name already exists
