import {useRef, useState} from 'react';

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
      label: t('launch.quick_undo.take_snapshot'),
      handler: takeQuickSnapshot
    },
    {
      label: t('launch.quick_undo.revert_snapshot'),
      handler: undoQuickSnapshot
    }
  ];

  return (
    <Box sx={{m: 2}}>
      <Box sx={{m: 2}}>{t('launch.quick_snapshot.summary')}</Box>
      <Stack direction="row" justifyContent="center" spacing={1}>
        <FormControl size="small" sx={{minWidth: 120}}>
          <InputLabel id="snapshot-slot-label">{t('launch.quick_undo.slot')}</InputLabel>
          <Select
            label={t('launch.quick_undo.slot')}
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
            {confirming ? t('launch.quick_undo.confirm') : options[menuIndex].label}
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
                        key={option.label}
                        onClick={() => {
                          setConfirming(false);
                          setMenuIndex(index);
                          setMenuOpen(false);
                        }}
                        selected={index === menuIndex}
                      >
                        {option.label}
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
