import expect from 'expect';

import { marginLonghand, paddingLonghand, pxToRem } from './helpers';

describe('Styles - helper', () => {
  describe('#marginLonghand', () => {
    it('returns all sides by default', () => {
      expect(marginLonghand('10px')).toEqual({
        marginBottom: '10px',
        marginLeft: '10px',
        marginRight: '10px',
        marginTop: '10px',
      });
    });

    it('allows specifying margin sides', () => {
      expect(marginLonghand('5px', ['bottom', 'top'])).toEqual({
        marginBottom: '5px',
        marginTop: '5px',
      });
    });
  });

  describe('#paddingLonghand', () => {
    it('returns all sides by default', () => {
      expect(paddingLonghand('10px')).toEqual({
        paddingBottom: '10px',
        paddingLeft: '10px',
        paddingRight: '10px',
        paddingTop: '10px',
      });
    });

    it('allows specifying margin sides', () => {
      expect(paddingLonghand('auto', ['bottom', 'top'])).toEqual({
        paddingBottom: 'auto',
        paddingTop: 'auto',
      });
    });
  });

  describe('#pxToRem', () => {
    it('calculates rem from the pixel input', () => {
      expect(pxToRem(16)).toEqual('1rem');
      expect(pxToRem(8)).toEqual('0.5rem');
    });
  });
});

