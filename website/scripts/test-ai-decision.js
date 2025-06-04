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
    ];

    let topPosts = [];
    await ƒ.simultaneouslyForEach(posts, async (post)=>{
      let postClassification = await ƒ.decide(post, {
        'Top post': 'A social media post that is both (a) VERY interesting and (b) in reasonably good taste',
        'n/a': 'Anything else',
      });
      if (postClassification === 'Top post') {
        topPosts.push(post);
      }
    });//∞

    return topPosts;

  }


};

