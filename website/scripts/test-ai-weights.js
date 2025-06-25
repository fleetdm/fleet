module.exports = {


  friendlyName: 'Test ai weights',


  description: '',


  fn: async function () {

    sails.log('Running custom shell script... (`sails run test-ai-weights`)');

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

    let weighedPosts = await ƒ.simultaneouslyForEach(posts, async (post)=>{
      return {
        ...post,
        scoresByTopic: await ƒ.weigh(post, [
          'related to cats',
          'related to javascript',
          'A social media post that is both (a) VERY interesting and (b) in reasonably good taste'
        ])
      };
    });//∞

    return weighedPosts;

  }


};

