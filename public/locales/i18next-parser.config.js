module.exports = {
  defaultNamespace: 'grafana',

  // Adds changes only to en-US when extracting keys, every other language is provided by Crowdin
  locales: ['en-US'],

  input: ['../../public/**/*.{tsx,ts}', '!../../public/app/extensions/**/*', '../../packages/grafana-ui/**/*.{tsx,ts}'],

  output: './public/locales/$LOCALE/$NAMESPACE.json',

  sort: true,

  createOldCatalogs: false,

  failOnWarnings: true,

  verbose: false,
};
