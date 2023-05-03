#!/bin/bash

run(){
	
	local OPTIND

  #default values
	output="fleetd_installers"
	flags+="--disable-open-folder"

  #Read flags
	while getopts s:p:u:f:d:o:x flag
	 do
		case "${flag}" in
			f) #path to file containing team names. Must end with newline char.
				source=($OPTARG);;
			p) #types of installers to create. Pass an individual flag for each type
				types+=($OPTARG);;
			u) #Fleet server url
				url=($OPTARG);;
			f) #Additional flags to apply to `fleetctl package`
				flags+=($OPTARG);;
			d) #include Fleet Desktop
				flags+="--desktop";;
			o) #Directory for created packages
		    output=($OPTARG);;
      x) #Test only
        dry_run="--dry-run";;
		esac
	 done

	#Verify that passed file exists
	if !(test -f "$source")
		then
			echo "Source file not found"
			return
  fi

  #Set up output directory
	if !(test -d "$output")
		then
				mkdir $output
	fi

  #If no package type specified, generate all
  if [[ (-z $types ) || ($types == "all")]]
	 	then 
		  types=("deb" "pkg" "msi" "rpm")
	fi

  create_teams
}

create_teams(){
  #Loop over file contents and generate a secret for each team, then create the team and generate packages
	while IFS=",", read -r name
		do
		  secret=$(LC_ALL=C tr -dc A-Za-z0-9 </dev/random | head -c 24);
		  team_name=$name

		  create_team
		  generate_packages
		done < $source
}

create_team(){

  #Generate yml based on template provided

	cat <<EOF > config.yml
---
apiVersion: v1
kind: team
spec:
  team:
    name: ${team_name}
    secrets:
      - secret: ${secret}
EOF

  # Apply the new team to fleet
	echo "Adding $team_name team to Fleet"
	fleetctl apply -f config.yml $dry_run
	rm -f config.yml 
}

generate_packages(){

	echo "Generating installers for $team_name"  

  #Set up directory to hold installers for this team
  name_formatted=$(printf "$team_name" | tr '[:upper:]' '[:lower:]' | tr -s ' ' | tr ' ' '-')
	team_dir=$output/$name_formatted
  cwd=$(pwd)
  
  if !(test -d "$team_dir")
		then
				mkdir "$team_dir"
	fi
  
  cd "$team_dir"

  #In the team directory, create a package for each specified type
	for type in ${types[@]}
    do
      fleetctl package ${flags[@]} --type=$type --fleet-url=$url --enroll-secret=$secret 
      find . -type f -name 'fleet-osquery*' -exec mv -f {} fleetd-$name_formatted.$type ';'
    done
  
  

  cd "$cwd"
  
}

run "$@"


