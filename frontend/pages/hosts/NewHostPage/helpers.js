import select from 'select';

const removeSelectedText = () => {
  return global.window.getSelection().removeAllRanges();
};

export const copyText = (elementId) => {
  const element = global.document.querySelector(elementId);

  select(element);

  const canCopy = global.document.queryCommandEnabled('copy');

  if (!canCopy) {
    return false;
  }

  global.document.execCommand('copy');
  removeSelectedText();
  return true;
};

export default { copyText };
