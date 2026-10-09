

module.exports = {


  friendlyName: 'Regenerate android management policy schema',


  description: 'Rebuild the Android Management API Policy schema the configuration profile generator provides to the model.',



  inputs: {

    dry: {
      type: 'boolean',
      defaultsTo: false,
      description: 'Parse and report, but do not write the output files.'
    },

  },

  fn: async function ({dry}) {

    let path = require('path');
    let util = require('util');

    let outputPath = path.resolve(sails.config.appPath, 'profile-generator/schema/android-policy-schema.json');

    let discoveryDocument = await sails.helpers.http.get('https://androidmanagement.googleapis.com/$discovery/rest?version=v1')
    .intercept((err)=>{
      return new Error(`When requesting the Android Management API discovery document, an error occurred. Full error: ${util.inspect(err, {depth: 2})}`);
    });

    if(!_.isObject(_.get(discoveryDocument, 'schemas.Policy.properties'))) {
      throw new Error('The Android Management API discovery document did not contain a "schemas.Policy" object with properties.  Has the document changed shape?');
    }
    let allSchemas = discoveryDocument.schemas;

    // Policy's properties $ref dozens of other schemas (PasswordRequirements,
    // AdvancedSecurityOverrides, and so on), so saving Policy alone would leave half of its
    // fields as dangling references.  Walk $refs transitively to collect the real dependency set.
    let findSchemaNamesReferencedWithin = (node)=>{
      if(!_.isObject(node)) {
        return [];
      }
      return _.reduce(node, (namesSoFar, value)=>{
        return _.union(namesSoFar, findSchemaNamesReferencedWithin(value));
      }, _.isString(node.$ref) ? [node.$ref] : []);
    };

    let relevantSchemaNames = ['Policy'];
    let schemaNamesToVisit = findSchemaNamesReferencedWithin(allSchemas.Policy);
    while(schemaNamesToVisit.length > 0) {
      let schemaName = schemaNamesToVisit.shift();
      if(_.contains(relevantSchemaNames, schemaName) || !allSchemas[schemaName]) {
        continue;// Already collected, or not a schema in this document.
      }
      relevantSchemaNames.push(schemaName);
      schemaNamesToVisit = _.union(schemaNamesToVisit, findSchemaNamesReferencedWithin(allSchemas[schemaName]));
    }

    // Keep a property's first sentence, plus any later sentence about deprecation, replacement, or
    // precedence.  Several fields bury "Replaces x (deprecated)" or "If tetheringSettings is set, this
    // setting is ignored" after the first sentence, and those are how the model finds the field that
    // actually takes effect.
    //
    // Google opens many descriptions with a bare "Optional." or "Required.", which would otherwise be
    // the only sentence kept -- printingPolicy came through as just "Optional.".  Some sentences
    // also run together with no space ("Use passwordPolicies.Note:Complexity..."), so a period
    // directly followed by a capital letter is a boundary too.
    let prunePropertyDescription = (description)=>{
      let sentences = _.reject(description.split(/(?<=\.)\s+|(?<=[a-z]\.)(?=[A-Z])/), (sentence)=>{
        return /^(optional|required)\.?$/i.test(sentence.trim());
      });
      return _.union(
        _.first(sentences) ? [_.first(sentences)] : [],
        _.filter(_.rest(sentences), (sentence)=>{ return /deprecat|replaces|instead|ignored|no effect|overrid|\buse\b/i.test(sentence); })
      ).join(' ');
    };

    // The full closure is ~161 KB (~41k tokens), too large to put in a prompt, so keep only what
    // is needed to build a valid policy: each field's type, its nesting, and its allowed values.
    // Enum arrays are kept verbatim -- enum casing is the one Android mistake nothing catches
    // before the policy reaches Google.
    //
    // Deprecated fields are kept, flagged by Google's own `deprecated` property, rather than
    // filtered out: the model is told to use the replacement, and it can only name the replacement
    // if it can see both.  Matching "deprecat" in descriptions instead would also flag fields like
    // AdvancedSecurityOverrides.developerSettings, which only mention deprecating something else.
    let prunedSchemasByName = _.reduce(relevantSchemaNames, (prunedSoFar, schemaName)=>{
      prunedSoFar[schemaName] = {
        properties: _.mapValues(allSchemas[schemaName].properties || {}, (property)=>{
          let prunedProperty = _.pick(property, ['type', '$ref', 'format', 'enum', 'deprecated']);
          if(property.items) {
            prunedProperty.items = _.pick(property.items, ['$ref', 'type', 'format', 'enum']);
          }
          if(property.description) {
            prunedProperty.description = prunePropertyDescription(property.description);
          }
          return prunedProperty;
        })
      };
      return prunedSoFar;
    }, {});

    let summary = `${relevantSchemaNames.length} schemas, ${Math.round(JSON.stringify(prunedSchemasByName).length / 1024)} KB`;
    if(dry) {
      sails.log(`Dry run: would have saved the Android Management API Policy schema to ${outputPath} (${summary}).`);
      return;
    }

    await sails.helpers.fs.writeJson.with({
      destination: outputPath,
      json: prunedSchemasByName,
      force: true
    });

    sails.log(`Android Management API Policy schema saved to ${outputPath} (${summary}).  Commit this file.`);

  }


};
