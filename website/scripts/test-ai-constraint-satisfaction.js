module.exports = {


  friendlyName: 'Test ai constraint satisfaction',


  description: '',


  fn: async function() {

    sails.log('Running custom shell script... (`sails run test-ai-constraint-satisfaction`)');

    let seatingChart = {
      elevenTop1: [ 'Rachael McNeil', 'Mike McNeil', 'Andrew Peterson', 'Ally Peterson', 'Tina Morales', 'Luke Morales', 'Becky Simon', 'Charlie Simon', ],
      elevenTop2: [ 'Ella Thompson', 'Jack Thompson', 'Laura Kim', 'Daniel Kim', 'Samantha Ortiz', 'Victor Ortiz', 'Annie Benson', 'Matt Benson', 'Michelle Reeves', 'Oscar Reeves', 'Pamela Frost' ],
      tenTop4: [ 'Ava Pruitt', 'Mason Pruitt', 'Harper Sloan', 'Logan Sloan', 'Peyton Sellers', 'Griffin Sellers', 'Kayla Lowe', 'Trevor Lowe', 'Eliza Pratt', 'Dean Pratt' ],
      //…etc
    };

    let newSeatingChart = await ƒ.satisfy(seatingChart, [
      'People with the same last name are married and should sit together.',
      'No table can have fewer than 8 people seated at it.'
    ], [
      'Add another, special 2-person table for the bride and groom, Ally and Andrew Peterson, and move them to it'
    ]);

    return newSeatingChart;

  }


};
