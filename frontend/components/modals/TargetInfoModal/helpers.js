export const headerClassName = (target) => {
  const { label, target_type: targetType } = target;

  if (label.toLowerCase() === 'all hosts') {
    return 'kolidecon-all-hosts';
  }

  return targetType === 'hosts' ? 'kolidecon-single-host' : 'kolidecon-label';
};

export default { headerClassName };
