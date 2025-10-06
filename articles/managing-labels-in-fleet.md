# Labels

![Managing labels in Fleet](../website/assets/images/articles/managing-labels-in-fleet-1600x900@2x.png)


Labels in Fleet provide a powerful way to scope profiles to specific hosts. This guide will walk you
through managing labels using the Fleet web UI. Labels can be created manually by selecting specific
hosts or dynamically using queries or host vitals. Dynamic labels are applied to hosts that match
either the query
or the host vitals criteria, while manual labels are assigned to hosts you select.


### Managing labels

To access and manage labels in Fleet, navigate to the Labels page by clicking on the user menu at the top
right of the window, then selecting "Labels".

### Filtering hosts by label

There are two ways to filter hosts by label:

* **From the Labels page**: Hover over the row of the label whose hosts you want to see, select the
"Actions" menu, and select "View all hosts" from the dropdown.

* **From the Hosts page**: Select the "Filter by platform or
label" drop-down menu, then select the label you want to filter by.

### Adding a label

To add a new label:

1. **Access the new label form**:
   * From the [Labels page](#managing-labels), click the "Add label" button, or
   * From the Hosts page, select the "Filter by platform or label" drop-down menu, then select the "Add
     label +" button.
2. **On the new label form**:
    1. **Enter a label name**: You may also provide an optional description for the label.
    2. **Choose label type**: You will be prompted to choose between "Dynamic", "Manual" or "Host vitals" label creation.
        1. **Dynamic**: Build your query and select the platforms to which this label applies.
        2. **Manual**: Select the hosts to which you want to apply this label.
        3. **Host vitals**: Select an attribute from the "Label criteria" dropdown, and enter a value that each host in the label should match for that attribute.
    3. **Save the label**: Click the "Save" button to create your label.


### Editing a label

To edit an existing label:

1. **Access the edit label form**:
  * **From the Labels page**:
    1. Hover over the row of the label you want to edit.
    2. Select the "Actions" menu.
    3. Select "Edit"
  * **From  the Hosts page**:
    1. [Filter by the desired label](#filtering-hosts-by-label).
    2. Click the pencil icon: A pencil icon will appear next to the label if it is editable. Clicking this icon allows you to edit the label.
2. **Edit the label details**: For manually applied labels, you can change the name, description,
   and selected hosts. For dynamically applied labels, you can view the query.  Host vitals labels
   cannot be edited at this time.

> **Update restrictions**: To change the query or platforms a dynamic label targets, you must delete the existing label and create a new one. Once set, label queries and platforms are immutable.


### Using the REST API for Labels

Fleet also provides a REST API to manage labels programmatically. The API allows you to add, update, retrieve, list, and delete labels. Find detailed documentation on Fleet's [REST API here](https://fleetdm.com/docs/rest-api/rest-api#labels).


### Managing labels with `fleetctl`

Fleet's command line tool, `fleetctl` will also allow you to list and manage labels. While managing labels with `fleetctl` is beyond the scope of this guide, you can list all labels using the following command:

```bash

fleetctl get labels

```

> Learn more about [`fleetctl` CLI](https://fleetdm.com/docs/using-fleet/fleetctl-cli).


#### Additional Information



* **Targeting extensions with labels**: Labels can also [target extensions to specific hosts](https://fleetdm.com/docs/configuration/agent-configuration#targeting-extensions-with-labels).


### Conclusion

Using labels in Fleet enhances your ability to effectively manage and scope profiles to specific hosts. Whether you prefer to manage labels through the web UI or programmatically via the REST API, Fleet provides the flexibility and control you need. For more information on using Fleet, please refer to the [Fleet documentation](https://fleetdm.com/docs) and [guides](https://fleetdm.com/guides).


<meta name="articleTitle" value="Managing labels in Fleet">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-06-30">
<meta name="articleImageUrl" value="../website/assets/images/articles/managing-labels-in-fleet-1600x900@2x.png">
<meta name="description" value="This guide will walk you through managing labels using the Fleet web UI.">
