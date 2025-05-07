module.exports = {


  friendlyName: 'Test ai decision',


  description: '',


  fn: async function () {

    sails.log('Running custom shell script... (`sails run test-ai-decision`)');

    let posts = [
      {
        id: 1,
        author: 'mikermcneil',
        tweet: 'I fed this one stray cat and now I have 20 stray cats coming to my house',
      },
      {
        id: 2,
        author: 'fancydoilies',
        tweet: 'My cat is named Rory'
      },
      {
        id: 3,
        author: 'koo',
        tweet: 'Sails.js is the best JavaScript framework'
      },
      {
        id: 4,
        author: 'koo',
        tweet: 'The 4th annual SailsConf is coming up in May in Abuja!'
      },
    ];

    let topPosts = [];
    await ƒ.simultaneouslyForEach(posts, async (post)=>{
      let postClassification = await ƒ.decide(post, [
        {
          predicate: 'A social media post that is both (a) interesting and (b) in reasonably good taste',
          value: 'Top post'
        }, {
          predicate: 'Anything else',
          value: 'n/a'
        },
      ]);
      if (postClassification === 'Top post') {
        topPosts.push(post);
      }
    });//∞

    return topPosts;

  }


};

