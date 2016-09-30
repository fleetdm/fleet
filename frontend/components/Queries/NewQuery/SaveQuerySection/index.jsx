import React, { PropTypes } from 'react';
import radium from 'radium';
import componentStyles from './styles';
import Slider from '../../../buttons/Slider';

const SaveQuerySection = ({ onToggleSaveQuery, saveQuery }) => {
  const {
    saveQuerySection,
    saveTextWrapper,
    saveWrapper,
    sliderTextDontSave,
    sliderTextSave,
  } = componentStyles;

  return (
    <section style={saveQuerySection}>
      <div style={saveTextWrapper}>
        <p>Save Query & Results For Later?</p>
        <small>For certain types of queries, like one that targets many hosts or one you plan to reuse frequently, we suggest saving the query & results. This allows you to set some advanced options, view the results later, and share with other users</small>
      </div>
      <div style={saveWrapper}>
        <span style={sliderTextDontSave(saveQuery)}>Dont save</span>
        <Slider onClick={onToggleSaveQuery} engaged={saveQuery} />
        <span style={sliderTextSave(saveQuery)}>Save</span>
      </div>
    </section>
  );
};

SaveQuerySection.propTypes = {
  onToggleSaveQuery: PropTypes.func,
  saveQuery: PropTypes.bool,
};

export default radium(SaveQuerySection);
