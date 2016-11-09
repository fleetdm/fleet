export const headerClassName = (target) => {
  const { display_text: displayText, target_type: targetType } = target;

  if (displayText.toLowerCase() === 'all hosts') {
    return 'kolidecon-all-hosts';
  }

  return targetType === 'hosts' ? 'kolidecon-single-host' : 'kolidecon-label';
};

export default { headerClassName };
