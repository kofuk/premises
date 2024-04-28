import React, {useRef, useState} from 'react';

import {useTranslation} from 'react-i18next';

import {ArrowDropDown as ArrowDropDownIcon} from '@mui/icons-material';
import {Button, ButtonGroup, ClickAwayListener, FormControl, Grow, InputLabel, MenuItem, MenuList, Paper, Popper, Select, Stack} from '@mui/material';
import {Box} from '@mui/system';

import {takeQuickSnapshot, undoQuickSnapshot} from '@/api';

const QuickUndo = () => {
  const [t] = useTranslation();
  const [selectedSlot, setSelectedSlot] = useState(0);
  const [menuOpen, setMenuOpen] = useState(false);
  const anchorRef = useRef<HTMLDivElement>(null);
  const [menuIndex, setMenuIndex] = useState(0);
  const [confirming, setConfirming] = useState(false);

  const handleClick = () => {
    if (!confirming) {
      setConfirming(true);
      return;
    }

    (async () => {
      try {
        await options[menuIndex].handler({slot: selectedSlot});
      } finally {
        setConfirming(false);
      }
    })();
  };

  const options = [
    {
      textKey: 'take_snapshot',
      handler: takeQuickSnapshot
    },
    {
      textKey: 'revert_snapshot',
      handler: undoQuickSnapshot
    }
  ];

  return (
    <Box sx={{m: 2}}>
      <Box sx={{m: 2}}>{t('snapshot_description')}</Box>
      <Stack direction="row" justifyContent="center" spacing={1}>
        <FormControl size="small" sx={{minWidth: 120}}>
          <InputLabel id="snapshot-slot-label">{t('snapshot_slot')}</InputLabel>
          <Select
            label={t('snapshot_slot')}
            labelId="snapshot-label-id"
            onChange={(e) => setSelectedSlot(parseInt(e.target.value as string))}
            value={selectedSlot}
          >
            {[0, 1, 2, 3, 4, 5, 6, 7, 8, 9].map((slot) => (
              <MenuItem key={`slot-${slot}`} selected={selectedSlot == slot} value={slot}>
                {`${slot}`}
              </MenuItem>
            ))}
          </Select>
        </FormControl>

        <ButtonGroup ref={anchorRef} variant="contained">
          <Button onClick={handleClick} type="button">
            {t(confirming ? 'snapshot_confirm' : options[menuIndex].textKey)}
          </Button>
          <Button onClick={() => setMenuOpen(!menuOpen)} size="small">
            <ArrowDropDownIcon />
          </Button>
        </ButtonGroup>
        <Popper anchorEl={anchorRef.current} disablePortal open={menuOpen} popperOptions={{strategy: 'fixed'}} transition>
          {({TransitionProps, placement}) => (
            <Grow
              {...TransitionProps}
              style={{
                transformOrigin: placement === 'bottom' ? 'center top' : 'center bottom'
              }}
            >
              <Paper>
                <ClickAwayListener onClickAway={() => setMenuOpen(false)}>
                  <MenuList autoFocusItem>
                    {options.map((option, index) => (
                      <MenuItem
                        key={option.textKey}
                        onClick={() => {
                          setConfirming(false);
                          setMenuIndex(index);
                          setMenuOpen(false);
                        }}
                        selected={index === menuIndex}
                      >
                        {`${t(option.textKey)}`}
                      </MenuItem>
                    ))}
                  </MenuList>
                </ClickAwayListener>
              </Paper>
            </Grow>
          )}
        </Popper>
      </Stack>
    </Box>
  );
};

export default QuickUndo;
