import React, {useEffect, useState} from 'react';
import KeyIcon from '@mui/icons-material/Key';
import CloseIcon from '@mui/icons-material/Close';
import PasswordIcon from '@mui/icons-material/Password';
import {
    DialogActions,
    DialogContent,
    DialogTitle,
    Dialog,
    IconButton,
    ButtonGroup,
    Tooltip,
    Box,
    Stack,
    Button,
    Card,
    Typography,
    CardContent,
    TextField,
    Snackbar
} from '@mui/material';
import {LoadingButton} from '@mui/lab';
import {Helmet, HelmetProvider} from 'react-helmet-async';

import '../i18n';
import {t} from 'i18next';
import {encodeBuffer, decodeBuffer} from '../base64url';
import {useNavigate} from 'react-router-dom';

interface WebAuthnLoginProps {
    setFeedback: (feedback: string) => void;
    switchToPassword: () => void;
}

const WebAuthnLogin: React.FC<WebAuthnLoginProps> = (props: WebAuthnLoginProps) => {
    const {setFeedback, switchToPassword} = props;

    const [username, setUsername] = useState('');
    const [loggingIn, setLoggingIn] = useState(false);

    const navigate = useNavigate();

    const handleWebAuthn = async () => {
        const params = new URLSearchParams();
        params.append('username', username);

        let beginResp: any = await fetch('/login/hardwarekey/begin', {
            method: 'post',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded'
            },
            body: params.toString()
        }).then((resp) => resp.json());

        if (!beginResp['success']) {
            setLoggingIn(false);
            setFeedback(beginResp['reason']);
            return;
        }

        const options = beginResp['options'];

        options.publicKey.challenge = decodeBuffer(options.publicKey.challenge);
        if (options.publicKey.allowCredentials) {
            for (let i = 0; i < options.publicKey.allowCredentials.length; i++) {
                options.publicKey.allowCredentials[i].id = decodeBuffer(options.publicKey.allowCredentials[i].id);
            }
        }

        const publicKeyCred = (await navigator.credentials.get(options)) as PublicKeyCredential;
        const rawId = publicKeyCred.rawId;
        const {authenticatorData, clientDataJSON, signature, userHandle} = publicKeyCred.response as AuthenticatorAssertionResponse;

        const finishResp: any = await fetch('/login/hardwarekey/finish', {
            method: 'post',
            body: JSON.stringify({
                id: publicKeyCred.id,
                rawId: encodeBuffer(rawId),
                type: publicKeyCred.type,
                response: {
                    authenticatorData: encodeBuffer(authenticatorData),
                    clientDataJSON: encodeBuffer(clientDataJSON),
                    signature: encodeBuffer(signature),
                    userHandle: encodeBuffer(userHandle!!)
                }
            })
        }).then((resp) => resp.json());

        if (!finishResp['success']) {
            setLoggingIn(false);
            setFeedback(finishResp['reason']);
            return;
        }

        navigate('/launch', {replace: true});

        setLoggingIn(false);
    };

    const login = () => {
        setLoggingIn(true);

        handleWebAuthn().catch(() => {
            setFeedback(t('passwordless_login_error'));
        });
    };

    return (
        <form
            onSubmit={(e) => {
                e.preventDefault();
                login();
            }}
        >
            <Stack spacing={2}>
                <TextField
                    variant="outlined"
                    label={t('username')}
                    autoComplete="username"
                    type="text"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    fullWidth
                />
                <Stack direction="row" justifyContent="end" sx={{mt: 1}}>
                    <ButtonGroup disabled={loggingIn} variant="contained" aria-label="outlined primary button group">
                        <Tooltip title="Use password">
                            <Button aria-label="password login" startIcon={<PasswordIcon />} type="button" onClick={() => switchToPassword()} />
                        </Tooltip>
                        <LoadingButton loading={loggingIn} variant="contained" type="submit">
                            {t('login')}
                        </LoadingButton>
                    </ButtonGroup>
                </Stack>
            </Stack>
        </form>
    );
};

interface PasswordLoginProps {
    setFeedback: (feedback: string) => void;
    switchToSecurityKey: () => void;
}

const PasswordLogin: React.FC<PasswordLoginProps> = (props: PasswordLoginProps) => {
    const {setFeedback, switchToSecurityKey} = props;

    const [loggingIn, setLoggingIn] = useState(false);
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');

    const [openResetPasswordDialog, setOpenResetPasswordDialog] = useState(false);

    const navigate = useNavigate();

    const login = () => {
        setLoggingIn(true);

        const params = new URLSearchParams();
        params.append('username', username);
        params.append('password', password);

        fetch('/login', {
            method: 'post',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded'
            },
            body: params.toString()
        })
            .then((resp) => resp.json())
            .then((resp) => {
                if (resp['success']) {
                    if (resp['needsChangePassword']) {
                        setOpenResetPasswordDialog(true);
                        return;
                    }

                    setLoggingIn(false);
                    navigate('/launch', {replace: true});
                    return;
                }
                setLoggingIn(false);
                setFeedback(resp['reason']);
            });
    };

    const [newPassword, setNewPassword] = useState('');
    const [newPasswordConfirm, setNewPasswordConfirm] = useState('');

    const changePassword = () => {
        const params = new URLSearchParams();
        params.append('username', username);
        params.append('password', newPassword);

        fetch('/login/reset-password', {
            method: 'post',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded'
            },
            body: params.toString()
        })
            .then((resp) => resp.json())
            .then((resp) => {
                if (resp['success']) {
                    location.reload();
                    return;
                }
                setFeedback(resp['reason']);
            });
    };

    return (
        <>
            <form
                onSubmit={(e) => {
                    e.preventDefault();
                    login();
                }}
            >
                <Stack spacing={2}>
                    <TextField
                        variant="outlined"
                        label={t('username')}
                        autoComplete="username"
                        type="text"
                        value={username}
                        onChange={(e) => setUsername(e.target.value)}
                        fullWidth
                    />
                    <TextField
                        variant="outlined"
                        label={t('password')}
                        autoComplete="password"
                        type="password"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        fullWidth
                    />
                    <Stack direction="row" justifyContent="end" sx={{mt: 1}}>
                        <ButtonGroup disabled={loggingIn} variant="contained" aria-label="outlined primary button group">
                            <Tooltip title="Use security key">
                                <Button aria-label="security key" startIcon={<KeyIcon />} type="button" onClick={() => switchToSecurityKey()} />
                            </Tooltip>
                            <LoadingButton loading={loggingIn} variant="contained" type="submit">
                                {t('login')}
                            </LoadingButton>
                        </ButtonGroup>
                    </Stack>
                </Stack>
            </form>
            <Dialog open={openResetPasswordDialog}>
                <DialogTitle>{t('set_password_title')}</DialogTitle>
                <form
                    onSubmit={(e) => {
                        e.preventDefault();
                        changePassword();
                    }}
                >
                    <DialogContent>
                        <Stack spacing={2}>
                            <TextField
                                label={t('change_password_new')}
                                type="password"
                                autoComplete="new-password"
                                value={newPassword}
                                onChange={(e) => setNewPassword(e.target.value)}
                                fullWidth
                            />
                            <TextField
                                label={t('change_password_confirm')}
                                type="password"
                                autoComplete="new-password"
                                value={newPasswordConfirm}
                                onChange={(e) => setNewPasswordConfirm(e.target.value)}
                                fullWidth
                            />
                        </Stack>
                    </DialogContent>
                    <DialogActions>
                        <Button disabled={!(newPassword.length != 0 && newPassword == newPasswordConfirm)} type="submit">
                            {t('set_password_submit')}
                        </Button>
                    </DialogActions>
                </form>
            </Dialog>
        </>
    );
};

const LoginApp = () => {
    const [feedback, setFeedback] = useState('');
    const [loginMethod, setLoginMethod] = useState('password');

    const navigate = useNavigate();

    useEffect(() => {
        fetch('/api/current-user')
            .then((resp) => resp.json())
            .then((resp) => {
                if (resp['success']) {
                    navigate('/launch', {replace: true});
                }
            });
    }, []);

    return (
        <Box display="flex" justifyContent="center">
            <Card sx={{minWidth: 350, p: 3, mt: 5}}>
                <CardContent>
                    <Typography variant="h4" component="h1" sx={{mb: 3}}>
                        {t('title_login')}
                    </Typography>
                    <Snackbar
                        anchorOrigin={{vertical: 'top', horizontal: 'center'}}
                        open={feedback.length > 0}
                        autoHideDuration={10000}
                        onClose={() => setFeedback('')}
                        message={feedback}
                        action={
                            <>
                                <IconButton aria-label="close" color="inherit" sx={{p: 0.5}} onClick={() => setFeedback('')}>
                                    <CloseIcon />
                                </IconButton>
                            </>
                        }
                    />
                    {loginMethod === 'password' ? (
                        <PasswordLogin setFeedback={setFeedback} switchToSecurityKey={() => setLoginMethod('webauthn')} />
                    ) : (
                        <WebAuthnLogin setFeedback={setFeedback} switchToPassword={() => setLoginMethod('password')} />
                    )}
                </CardContent>
            </Card>
            <Helmet>
                <title>{t('title_login')}</title>
            </Helmet>
        </Box>
    );
};

export default LoginApp;
