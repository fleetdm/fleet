#!/bin/bash

run(){
	
	local OPTIND

  #default values
	output="fleet_osquery_packages"
  header=false
	flags+="--disable-open-folder"

  #Read flags
	while getopts s:hc:n:p:u:f:d:o:x flag
	 do
		case "${flag}" in
			s) #path to csv file containing team names. Must end with newline char.
				source=($OPTARG);;
			h) #indicates whether there is a header included in the file. default false;
				header=true;;
			c) #columns in csv file. if not provided, assumes a simple list of names separated by line breaks. not needed if header is provided.
        set -f 
        IFS=',' 
        columns+=($OPTARG) ;; # use the split+glob operator
			n) #column to use as team name
				name_column=($OPTARG);;
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
      x) #Dry run
        dry_run="--dry-run";;
		esac
	 done


	#Verify that passed file exists
	if !(test -f "$source")
		then
			echo "Source file not found"
			return
  fi

  #Set up for simple file with single column and no header
  if [[ (-z $columns ) && ($header == false)]]
	 	then 
    echo "here"
		columns="name"
    name_column="name"
    echo ${columns[@]}
    echo $name_column
	fi

  #Set up output directory
	if !(test -d $output)
		then
				mkdir $output
	fi

  create_teams
}

create_teams(){
   
  # If a header is provided, parse the first line of the file to retrieve columns and skip first line of file when creating teams
  if ($header == true )
	  then
	    i=1
		  columns=($(head -n1 $source | tr , '\n'))
	  else
	    i=0
	fi

  #Loop over file contents and generate a secret for each team, then create the team and generate packages
	while IFS=",", read -r ${columns[@]}
		do
		  test $i -eq 1 && ((i=i+1)) && continue

		  secret=$(LC_ALL=C tr -dc A-Za-z0-9 </dev/random | head -c 24);
		  name=${!name_column}

		  create_team
		  generate_packages
		done < $source
}

create_team(){

  #Generate yml based on template provided

	cat <<EOF > final.yml
apiVersion: v1
kind: team
spec:
  team:
    name: ${name}
    secrets:
      - secret: ${secret}
EOF

  # Apply the new team to fleet
	echo "Adding $name team to Fleet"
	fleetctl apply -f final.yml $dry_run
	rm -f final.yml temp.yml
}

generate_packages(){

	echo "Generating installers for $name"  

  #Set up directory to hold installers for this team
  name_formatted=$(printf "$name" | tr '[:upper:]' '[:lower:]' | tr -s ' ' | tr ' ' '_')
	team_dir=$output/$name_formatted
  mkdir $team_dir

  #set up variables for navigation into and back out of team directory
  cwd=$(pwd)
  cd $team_dir

  #In the team directory, create a package for each specified type
	for type in ${packages[@]}
  do
		(cd -- "$teamdir" && fleetctl package --type=$type --fleet-url=$url --enroll-secret=$secret ${flags[@]})
	done
  		find . -type f -name 'fleet-osquery*' -exec mv -f {} fleet_osquery_$name_formatted.$type ';'
  
  #Return to current location
  cd $cwd
}

run "$@"


