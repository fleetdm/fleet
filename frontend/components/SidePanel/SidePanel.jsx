import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { isEqual, last } from 'lodash';
import { push } from 'react-router-redux';
import radium, { StyleRoot } from 'radium';
import { activeTabFromPathname, activeSubTabFromPathname } from './helpers';
import componentStyles from './styles';
import kolideLogo from '../../../assets/images/kolide-logo.svg';
import navItems from './navItems';
import './styles.scss';

class SidePanel extends Component {
  static propTypes = {
    config: PropTypes.shape({
      org_logo_url: PropTypes.string,
      org_name: PropTypes.name,
    }),
    dispatch: PropTypes.func,
    pathname: PropTypes.string,
    user: PropTypes.object,
  };

  constructor (props) {
    super(props);

    const { pathname, user: { admin } } = this.props;

    this.userNavItems = navItems(admin);

    const activeTab = activeTabFromPathname(this.userNavItems, pathname);
    const activeSubItem = activeSubTabFromPathname(activeTab, pathname);

    this.state = {
      activeTab,
      activeSubItem,
      showSubItems: false,
    };
  }

  componentWillReceiveProps (nextProps) {
    if (isEqual(nextProps, this.props)) return false;

    const { pathname } = nextProps;

    const activeTab = activeTabFromPathname(this.userNavItems, pathname);
    const activeSubItem = activeSubTabFromPathname(activeTab, pathname);

    this.setState({
      activeTab,
      activeSubItem,
    });

    return false;
  }

  setActiveSubItem = (activeSubItem) => {
    return (evt) => {
      evt.preventDefault();

      const { dispatch } = this.props;
      const { path: { location: tabLocation } } = activeSubItem;

      if (!tabLocation) return false;

      dispatch(push(tabLocation));
      return false;
    };
  }

  setActiveTab = (activeTab) => {
    return (evt) => {
      evt.preventDefault();

      const { pathname, dispatch } = this.props;
      const activeSubItem = activeSubTabFromPathname(activeTab, pathname);
      const { path: { location: tabLocation } } = activeSubItem;

      this.setState({ activeTab, activeSubItem });
      if (tabLocation) return dispatch(push(tabLocation));

      return false;
    };
  }

  setSubNavClass = (showSubItems) => {
    return showSubItems ? 'sub-nav sub-nav--expanded' : 'sub-nav';
  }

  toggleShowSubItems = (showSubItems) => {
    return (evt) => {
      evt.preventDefault();

      this.setState({ showSubItems });

      return false;
    };
  }

  renderHeader = () => {
    const {
      config: {
        org_name: orgName,
      },
      user: {
        enabled,
        username,
      },
    } = this.props;
    const {
      companyLogoStyles,
      headerStyles,
      orgNameStyles,
      usernameStyles,
      userStatusStyles,
      orgChevronStyles,
    } = componentStyles;

    return (
      <header style={headerStyles}>
        <img
          alt="Company logo"
          src={kolideLogo}
          style={companyLogoStyles}
        />
        <h1 style={orgNameStyles}>{orgName}</h1>
        <div style={userStatusStyles(enabled)} />
        <h2 style={usernameStyles}>{username}</h2>
        <i style={orgChevronStyles} className="kolidecon-chevrondownbold" />
      </header>
    );
  }

  renderNavItem = (navItem, lastChild) => {
    const { activeTab = {} } = this.state;
    const { icon, name, subItems } = navItem;
    const active = activeTab.name === name;
    const {
      iconStyles,
      navItemBeforeStyles,
      navItemNameStyles,
      navItemStyles,
      navItemWrapperStyles,
    } = componentStyles;
    const { renderSubItems, setActiveTab } = this;

    return (
      <div style={navItemWrapperStyles(lastChild)} key={`nav-item-${name}`}>
        {active && <div style={navItemBeforeStyles} />}
        <li
          key={name}
          onClick={setActiveTab(navItem)}
          style={navItemStyles(active)}
        >
          <div style={{ position: 'relative' }}>
            <i className={icon} style={iconStyles} />
            <span style={navItemNameStyles}>
              {name}
            </span>
          </div>
        </li>
        {active && renderSubItems(subItems)}
      </div>
    );
  }

  renderNavItems = () => {
    const { renderNavItem, userNavItems } = this;
    const { navItemListStyles } = componentStyles;
    const { user: { admin } } = this.props;

    return (
      <ul style={navItemListStyles}>
        {userNavItems.map((navItem, index, collection) => {
          const lastChild = admin && isEqual(navItem, last(collection));
          return renderNavItem(navItem, lastChild);
        })}
      </ul>
    );
  }

  renderSubItem = (subItem) => {
    const { activeSubItem } = this.state;
    const { name, path } = subItem;
    const active = activeSubItem === subItem;
    const { setActiveSubItem } = this;
    const {
      subItemBeforeStyles,
      subItemStyles,
      subItemLinkStyles,
    } = componentStyles;

    return (
      <div
        key={`sub-item-${name}`}
        style={{ position: 'relative' }}
      >
        {active && <div style={subItemBeforeStyles} />}
        <li
          key={name}
          onClick={setActiveSubItem(subItem)}
          style={subItemStyles(active)}
        >
          <span to={path.location} style={subItemLinkStyles(active)}>{name}</span>
        </li>
      </div>
    );
  }

  renderSubItems = (subItems) => {
    const { subItemListStyles, subItemsStyles } = componentStyles;
    const { renderCollapseSubItems, renderSubItem, setSubNavClass } = this;
    const { showSubItems } = this.state;

    if (!subItems.length) return false;

    return (
      <div className={setSubNavClass(showSubItems)} style={subItemsStyles}>
        <ul style={subItemListStyles(showSubItems)}>
          {subItems.map(subItem => {
            return renderSubItem(subItem);
          })}
        </ul>
        {renderCollapseSubItems()}
      </div>
    );
  }

  renderCollapseSubItems = () => {
    const { toggleShowSubItems } = this;
    const { showSubItems } = this.state;
    const { collapseSubItemsWrapper } = componentStyles;
    const iconName = showSubItems ? 'kolidecon-chevronleftbold' : 'kolidecon-chevronrightbold';

    return (
      <div style={collapseSubItemsWrapper} onClick={toggleShowSubItems(!showSubItems)}>
        <i className={iconName} />
      </div>
    );
  }

  render () {
    const { navStyles } = componentStyles;
    const { renderHeader, renderNavItems } = this;

    return (
      <StyleRoot>
        <nav style={navStyles}>
          {renderHeader()}
          {renderNavItems()}
        </nav>
      </StyleRoot>
    );
  }
}

const ConnectedComponent = connect()(SidePanel);
export default radium(ConnectedComponent);
