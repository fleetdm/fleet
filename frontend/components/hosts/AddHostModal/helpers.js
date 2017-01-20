import select from 'select';

const removeSelectedText = () => {
  return global.window.getSelection().removeAllRanges();
};

export const copyText = (elementSelector) => {
  const { document } = global;

  const element = document.querySelector(elementSelector);
  element.querySelector('input').type = 'text';

  select(element);

  const canCopy = document.queryCommandEnabled('copy');

  if (!canCopy) {
    return false;
  }

  document.execCommand('copy');
  element.querySelector('input').type = 'password';
  removeSelectedText();
  return true;
};

export default { copyText };
