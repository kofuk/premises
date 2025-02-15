import React, {useState} from 'react';

import {useForm} from 'react-hook-form';
import {useTranslation} from 'react-i18next';
import {toast} from 'react-toastify';

import {
  Delete as DeleteIcon,
  Download as DownloadIcon,
  Folder as FolderIcon,
  FolderOpen as FolderOpenIcon,
  Refresh as RefreshIcon,
  Upload as UploadIcon,
  Public as WorldIcon
} from '@mui/icons-material';
import {Button, ButtonGroup, IconButton, Stack, TextField, colors} from '@mui/material';
import {SimpleTreeView, TreeItem} from '@mui/x-tree-view';

import {APIError, createWorldDownloadLink, createWorldUploadLink, deleteWorld} from '@/api';
import {World, WorldGeneration} from '@/api/entities';
import {useAuth} from '@/utils/auth';

type Selection = {
  worldName: string;
  generationId: string;
};

export type ChangeEventHandler = (selection: Selection) => void;

type Props = {
  worlds: World[] | undefined;
  onChange?: ChangeEventHandler;
  selection?: Selection;
  refresh?: () => void;
};

const getWorldLabel = (worldGen: WorldGeneration): string => {
  const dateTime = new Date(worldGen.timestamp);
  const label = worldGen.gen.match(/[0-9]+-[0-9]+-[0-9]+ [0-9]+:[0-9]+:[0-9]+/)
    ? dateTime.toLocaleString()
    : `${worldGen.gen} (${dateTime.toLocaleString()})`;
  return label;
};

const WorldExplorer = ({worlds, selection, onChange, refresh}: Props) => {
  const [t] = useTranslation();

  const {accessToken} = useAuth();

  const handleSelectedItemsChange = (_event: React.SyntheticEvent, id: string | null) => {
    if (!id || !onChange) {
      return;
    }

    if (id.includes('/')) {
      const [worldName] = id.split('/');
      onChange({worldName, generationId: id});
    } else {
      onChange({worldName: id, generationId: '@/latest'});
    }
  };

  const handleDownload = async (generationId: string) => {
    try {
      const {url} = await createWorldDownloadLink(accessToken, {id: generationId});

      const a = document.createElement('a');
      a.href = url;
      a.target = '_blank';
      a.rel = 'noopener noreferrer';
      // `download` attribute for cross-origin URLs will be blocked in the most browsers, but it's not a problem for us.
      a.download = '';
      document.body.appendChild(a);
      a.click();
      a.remove();
    } catch (err) {
      if (err instanceof APIError) {
        toast.error(err.message);
      }
    }
  };

  const handleDeleteWorld = async (generationId: string) => {
    try {
      await deleteWorld(accessToken, {id: generationId});
    } catch (err) {
      if (err instanceof APIError) {
        toast.error(err.message);
      }
    }
    refresh?.();
  };

  const items = worlds?.map((world) => (
    <TreeItem key={world.worldName} itemId={world.worldName} label={world.worldName}>
      {world.generations.map((gen) => (
        <TreeItem
          key={gen.id}
          itemId={gen.id}
          label={
            <>
              {getWorldLabel(gen)}
              <IconButton
                onClick={(e) => {
                  e.stopPropagation();
                  handleDownload(gen.id);
                }}
              >
                <DownloadIcon />
              </IconButton>
              <IconButton
                onClick={(e) => {
                  e.stopPropagation();
                  handleDeleteWorld(gen.id);
                }}
              >
                <DeleteIcon />
              </IconButton>
            </>
          }
        />
      ))}
    </TreeItem>
  ));
  const selectedItems = selection && (selection.generationId === '@/latest' ? selection.worldName : selection.generationId);

  const [uploadMode, setUploadMode] = useState(false);
  const [isUploading, setUploading] = useState(false);
  const {register, handleSubmit} = useForm();

  const handleUpload = async ({worldName}: any) => {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.zip,.tar.gz,.tar.zst';
    input.onchange = async () => {
      setUploading(true);

      const file = input.files?.[0];
      if (!file) {
        return;
      }

      try {
        const {url} = await createWorldUploadLink(accessToken, {worldName, mimeType: file.type});
        await fetch(url, {
          method: 'PUT',
          body: file
        });
      } catch (err) {
        if (err instanceof APIError) {
          toast.error(err.message);
        }
      } finally {
        setUploadMode(false);
        setUploading(false);
        refresh?.();
      }
    };
    input.click();
  };

  return (
    <Stack spacing={0.5}>
      <Stack alignSelf="end" direction="row" spacing={1}>
        {uploadMode && (
          <form onSubmit={handleSubmit(handleUpload)}>
            <ButtonGroup>
              <TextField
                defaultValue={selection?.worldName}
                disabled={isUploading}
                label={t('launch.world.name')}
                sx={{
                  '& .MuiInputBase-input': {height: 6},
                  '& .MuiInputLabel-root[data-shrink=false]': {fontSize: 12, transform: 'translate(14px, 12px) scale(1)'}
                }}
                variant="outlined"
                {...register('worldName', {
                  required: true,
                  validate: (val: string) => !val.includes('/') && !val.includes('\\') && !val.includes('@')
                })}
              />
              <Button loading={isUploading} type="submit" variant="outlined">
                <UploadIcon />
              </Button>
            </ButtonGroup>
          </form>
        )}
        <Stack alignSelf="end" direction="row" sx={{backgroundColor: colors.blue[100], px: 2, borderRadius: '50vh'}}>
          {refresh && (
            <IconButton onClick={refresh}>
              <RefreshIcon />
            </IconButton>
          )}
          <IconButton onClick={() => setUploadMode(!uploadMode)}>
            <UploadIcon />
          </IconButton>
        </Stack>
      </Stack>
      <SimpleTreeView
        checkboxSelection={true}
        onSelectedItemsChange={handleSelectedItemsChange}
        selectedItems={selectedItems}
        slots={{
          expandIcon: FolderIcon,
          collapseIcon: FolderOpenIcon,
          endIcon: WorldIcon
        }}
      >
        {items}
      </SimpleTreeView>
    </Stack>
  );
};

export default WorldExplorer;
