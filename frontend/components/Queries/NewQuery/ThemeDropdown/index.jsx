import React, { PropTypes } from 'react';
import radium from 'radium';
import 'brace/mode/sql';
import 'brace/theme/dreamweaver';
import 'brace/theme/cobalt';
import 'brace/theme/eclipse';
import 'brace/theme/github';
import 'brace/theme/idle_fingers';
import 'brace/theme/iplastic';
import 'brace/theme/katzenmilch';
import 'brace/theme/kr_theme';
import 'brace/theme/kuroir';
import 'brace/theme/merbivore';
import 'brace/theme/merbivore_soft';
import 'brace/theme/mono_industrial';
import 'brace/theme/monokai';
import 'brace/theme/solarized_light';
import 'brace/theme/sqlserver';
import 'brace/theme/tomorrow';
import componentStyles from './styles';

const ThemeDropdown = ({ onSelectChange, theme }) => {
  const { themeDropdownStyles } = componentStyles;
  return (
    <div style={themeDropdownStyles}>
      <span style={{ fontSize: '10px' }}>Editor Theme:</span>
      <select onChange={onSelectChange} style={themeDropdownStyles} value={theme}>
        <option value="kolide">Kolide</option>
        <option value="dreamweaver">Dreamweaver</option>
        <option value="cobalt">Cobalt</option>
        <option value="eclipse">Eclipse</option>
        <option value="github">Github</option>
        <option value="idle_fingers">Idle Fingers</option>
        <option value="iplastic">Iplastic</option>
        <option value="katzenmilch">Katzenmilch</option>
        <option value="kr_theme">KR Theme</option>
        <option value="kuroir">Kuroir</option>
        <option value="merbivore">Merbivore</option>
        <option value="merbivore_soft">Merbivore Soft</option>
        <option value="mono_industrial">Mono Industrial</option>
        <option value="monokai">Monokai</option>
        <option value="solarized_light">Solarized Light</option>
        <option value="sqlserver">SQL Server</option>
        <option value="tomorrow">Tomorrow</option>
      </select>
    </div>
  );
};

ThemeDropdown.propTypes = {
  onSelectChange: PropTypes.func,
  theme: PropTypes.string,
};

export default radium(ThemeDropdown);
