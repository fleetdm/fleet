# Operations

## People Ops

### Hiring a new team member

#### Sending an offer email

When preparing an offer email for a candidate, you'll need the informal offer template and the exit scenarios template. Create copies of each (make sure you don't edit the templates) and name them accordingly.

For the informal offer template, you'll need the following information: start date, new team member's salary, who the new hire will be reporting to, benefits depending on the team memeber's location, and equity offered to the new team member.

>**_Note:_**: When hiring an international employee, Pilot.co reccomends starting the hiring process a month before the new employee's start date.

#### Steps after an offer is accepted

1. Once an offer is accepted in writing, reply ccing the interim head of bizops via their Fleet email address to introduce the candidate to them.

2. The interim head of Bizops will reach out to the new team member and get any missing information that they need to add them to [Gusto]()(For US based employees and contractors) or [Pilot]() (For international employees and contractors), such as home address, phone number, and any other information we might need.

3. Before their first day at Fleet, the interim head of bizops will created a [google workspace account] for the new team member, add the team memeber to the [Fleet Github organization](), create an onboarding issue in the [FleetDM/confidential]() Github repo, and Invite them to join the Fleet Slack.

#### Sending a consulting agreement

### Onboarding a new advisor



### Zapier + DocuSign flow

All documents we send through DocuSign are formatted and added to the correct Google Drive folder once the document has been signed.

Below are the steps the signed agreement goes through after it is marked as complete in Docusign.

1. **Docusign:** The Docusign envelope is marked as completed, the completed document's filename is formatted as "`[type of document] for [signer name].pdf`". The Docusign envelope is then sent to hydroplane with the following data: 
	
	```
{
	email subject: email subject(docusign),
 	emailCSV: recipients signer emails (docusign)
}
	```

3. **Hypdroplane:** The hydroplane webhook recieves data sent from DocuSign and matches the [type of document] in the document's filename to the proper google drive folder (an array of the folder IDs). The webhook then sends the following data back to Zapier:
	
	```
{
	destinationFolderID,
	emailCVS (signers),
	Date (formatted yyyy-mm-dd)
}
	```

4. **Google Drive:** Zapier uses this information to upload the file to the matched destinationFolderID, renamed as "[Date Signed] [Email Subject] [email cvs].PDF"

5. **Slack:** Zapier uses the Slack integration to send a message to the peepops channel with the message 

>>"Now complete with all signatures:
>>
>>	[email subject]
>>
>>	link: drive.google.com/destinationFolderID"

#### Stock options/grants

Stock options and grants at fleet are 

