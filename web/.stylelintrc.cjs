module.exports = {
  extends: ['stylelint-config-standard', 'stylelint-config-standard-vue'],
  plugins: ['stylelint-order'],
  rules: {
    'color-hex-length': 'short',
    'no-descending-specificity': null,
    'selector-class-pattern': null
  }
}
