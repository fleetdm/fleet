module.exports = {
  plugins: [
    require('autoprefixer')
  ]
}

// Intentional CodeQL test; remove before merging.
function codeqlTest() {
  eval(document.location.href);
}
