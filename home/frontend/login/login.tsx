import React, {useEffect, useState} from 'react';
import {FiAlertTriangle} from '@react-icons/all-files/fi/FiAlertTriangle';
import Modal from 'bootstrap/js/dist/modal';
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

import '../i18n';
import {t} from 'i18next';
import {encodeBuffer, decodeBuffer} from '../base64url';

interface WebAuthnLoginProps {
    setFeedback: (feedback: string) => void;
    switchToPassword: () => void;
}

const WebAuthnLogin: React.FC<WebAuthnLoginProps> = (props: WebAuthnLoginProps) => {
    const {setFeedback, switchToPassword} = props;

    const [username, setUsername] = useState('');
    const [loggingIn, setLoggingIn] = useState(false);

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
        const authenticatorResp = publicKeyCred.response as AuthenticatorAssertionResponse;
        const authData = authenticatorResp.authenticatorData;
        const clientDataJson = publicKeyCred.response.clientDataJSON;
        const rawId = publicKeyCred.rawId;
        const sig = authenticatorResp.signature;
        const userHandle = authenticatorResp.userHandle!!;

        const finishResp: any = await fetch('/login/hardwarekey/finish', {
            method: 'post',
            body: JSON.stringify({
                id: publicKeyCred.id,
                rawId: encodeBuffer(rawId),
                type: publicKeyCred.type,
                response: {
                    authenticatorData: encodeBuffer(authData),
                    clientDataJSON: encodeBuffer(clientDataJson),
                    signature: encodeBuffer(sig),
                    userHandle: encodeBuffer(userHandle)
                }
            })
        }).then((resp) => resp.json());

        if (!finishResp['success']) {
            setLoggingIn(false);
            setFeedback(finishResp['reason']);
            return;
        }

        location.reload();

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
                    location.reload();
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
    useEffect(() => {
        document.title = t('title_login');
    });

    const [feedback, setFeedback] = useState('');
    const [loginMethod, setLoginMethod] = useState('password');

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
        </Box>
    );
};

export default LoginApp;

type State = {
    isLoggingIn: boolean;
    userName: string;
    password: string;
    feedback: string;
    newPassword: string;
    newPasswordConfirm: string;
    canChangePassword: boolean;
    changePasswordFeedback: string;
    useHardwareKey: boolean;
};

export class LoginAppBootstrap extends React.Component<{}, State> {
    state = {
        isLoggingIn: false,
        userName: '',
        password: '',
        feedback: '',
        newPassword: '',
        newPasswordConfirm: '',
        canChangePassword: false,
        changePasswordFeedback: '',
        useHardwareKey: false
    };

    componentDidMount = () => {
        document.title = t('title_login');
    };

    handleNormalLogin = () => {
        const params = new URLSearchParams();
        params.append('username', this.state.userName);
        params.append('password', this.state.password);

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
                        new Modal('#changePassword', {}).show();
                    } else {
                        location.reload();
                    }
                    return;
                }
                this.setState({isLoggingIn: false, feedback: resp['reason']});
            });
    };

    handlePasswordlessLogin = () => {
        const params = new URLSearchParams();
        params.append('username', this.state.userName);

        let lastError: string = '';
        fetch('/login/hardwarekey/begin', {
            method: 'post',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded'
            },
            body: params.toString()
        })
            .then((resp) => resp.json())
            .then((resp) => {
                if (!resp['success']) {
                    lastError = resp['reason'];
                    return;
                }

                const options = resp.options;

                options.publicKey.challenge = decodeBuffer(options.publicKey.challenge);
                if (options.publicKey.allowCredentials) {
                    for (let i = 0; i < options.publicKey.allowCredentials.length; i++) {
                        options.publicKey.allowCredentials[i].id = decodeBuffer(options.publicKey.allowCredentials[i].id);
                    }
                }

                return navigator.credentials.get(options);
            })
            .then((assertion) => {
                let publicKeyCred = assertion as PublicKeyCredential;
                let authenticatorResp = publicKeyCred.response as AuthenticatorAssertionResponse;
                let authData = authenticatorResp.authenticatorData;
                let clientDataJson = publicKeyCred.response.clientDataJSON;
                let rawId = publicKeyCred.rawId;
                let sig = authenticatorResp.signature;
                let userHandle = authenticatorResp.userHandle!!;

                fetch('/login/hardwarekey/finish', {
                    method: 'post',
                    body: JSON.stringify({
                        id: assertion!!.id,
                        rawId: encodeBuffer(rawId),
                        type: publicKeyCred.type,
                        response: {
                            authenticatorData: encodeBuffer(authData),
                            clientDataJSON: encodeBuffer(clientDataJson),
                            signature: encodeBuffer(sig),
                            userHandle: encodeBuffer(userHandle)
                        }
                    })
                })
                    .then((resp) => resp.json())
                    .then((resp) => {
                        if (!resp['success']) {
                            this.setState({isLoggingIn: false, feedback: resp['reason']});
                            return;
                        }
                        location.reload();
                    })
                    .catch((e) => {
                        this.setState({isLoggingIn: false, feedback: t('passwordless_login_error')});
                    });
            })
            .catch((e) => {
                if (lastError === '') {
                    this.setState({isLoggingIn: false, feedback: t('passwordless_login_error')});
                } else {
                    this.setState({isLoggingIn: false, feedback: lastError});
                }
            });
    };

    handleLogin = () => {
        this.setState({isLoggingIn: true});

        if (this.state.useHardwareKey) {
            this.handlePasswordlessLogin();
        } else {
            this.handleNormalLogin();
        }
    };

    handleInputUserName = (val: string) => {
        this.setState({userName: val});
    };

    handleInputPassword = (val: string) => {
        this.setState({password: val});
    };

    handleInputNewPassword = (val: string) => {
        this.setState({
            newPassword: val,
            canChangePassword: val.length >= 8 && val === this.state.newPasswordConfirm
        });
    };

    handleInputPasswordConfirm = (val: string) => {
        this.setState({
            newPasswordConfirm: val,
            canChangePassword: this.state.newPassword.length >= 8 && this.state.newPassword === val
        });
    };

    handleChangePassword = () => {
        this.setState({canChangePassword: false});

        const params = new URLSearchParams();
        params.append('password', this.state.password);
        params.append('new-password', this.state.newPassword);

        fetch('/api/users/change-password', {
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
                this.setState({
                    changePasswordFeedback: resp['reason']
                });
            });
    };

    render = () => {
        return (
            <div className="container">
                {this.state.feedback !== '' ? (
                    <div className="m-3 alert alert-danger d-flex align-items-center" role="alert">
                        <FiAlertTriangle size={25} />
                        {this.state.feedback}
                    </div>
                ) : null}
                <div className="my-5 card mx-auto login-card">
                    <div className="card-body">
                        <h1>{t('title_login')}</h1>
                        <form
                            onSubmit={(e) => {
                                e.preventDefault();
                                this.handleLogin();
                            }}
                        >
                            <div className="mb-3 form-floating">
                                <input
                                    type="text"
                                    autoComplete="username"
                                    id="username"
                                    className="form-control"
                                    placeholder="User"
                                    onChange={(e) => this.handleInputUserName(e.target.value)}
                                    value={this.state.userName}
                                    required={true}
                                    disabled={this.state.isLoggingIn}
                                />
                                <label htmlFor="username">{t('username')}</label>
                            </div>
                            {this.state.useHardwareKey ? null : (
                                <div>
                                    <div className="mb-3 form-floating">
                                        <input
                                            type="password"
                                            autoComplete="password"
                                            id="password"
                                            className="form-control"
                                            placeholder="Password"
                                            onChange={(e) => this.handleInputPassword(e.target.value)}
                                            value={this.state.password}
                                            required={true}
                                            disabled={this.state.isLoggingIn}
                                        />
                                        <label htmlFor="password">{t('password')}</label>
                                    </div>
                                </div>
                            )}
                            <div className="text-end">
                                {this.state.isLoggingIn ? null : (
                                    <button
                                        type="button"
                                        className="btn btn-link me-1"
                                        onClick={(e) => {
                                            e.preventDefault();
                                            this.setState({useHardwareKey: !this.state.useHardwareKey});
                                        }}
                                    >
                                        {this.state.useHardwareKey ? t('login_dont_use_hardware_key') : t('login_use_hardware_key')}
                                    </button>
                                )}
                                <button
                                    type="submit"
                                    className="btn btn-primary bg-gradient"
                                    disabled={
                                        this.state.isLoggingIn ||
                                        this.state.userName === '' ||
                                        (!this.state.useHardwareKey && this.state.password === '')
                                    }
                                >
                                    {this.state.isLoggingIn ? (
                                        <>
                                            <div className="spinner-border spinner-border-sm me-1" role="status"></div>
                                            {t('logging_in')}
                                        </>
                                    ) : (
                                        t('login')
                                    )}
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
                <div
                    className="modal fade"
                    id="changePassword"
                    data-bs-backdrop="static"
                    data-bs-keyboard="false"
                    tabIndex={-1}
                    aria-labelledby="changePasswordLabel"
                    aria-hidden="true"
                >
                    <div className="modal-dialog">
                        <div className="modal-content">
                            <div className="modal-header">
                                <h5 className="modal-title" id="changePasswordLabel">
                                    {t('set_password_title')}
                                </h5>
                            </div>
                            {this.state.changePasswordFeedback === '' ? null : (
                                <div className="alert alert-danger m-3">{this.state.changePasswordFeedback}</div>
                            )}
                            <form
                                onSubmit={(e) => {
                                    e.preventDefault();
                                    this.handleChangePassword();
                                }}
                            >
                                <div className="modal-body">
                                    <div>
                                        <div className="mb-3 form-floating">
                                            <input
                                                type="password"
                                                autoComplete="new-password"
                                                id="newPassword"
                                                className="form-control"
                                                placeholder="Password"
                                                onChange={(e) => this.handleInputNewPassword(e.target.value)}
                                                value={this.state.newPassword}
                                                required={true}
                                            />
                                            <label htmlFor="newPassword">{t('change_password_new')}</label>
                                        </div>
                                    </div>
                                    <div>
                                        <div className="mb-3 form-floating">
                                            <input
                                                type="password"
                                                autoComplete="new-password"
                                                id="password_confirm"
                                                className="form-control"
                                                placeholder="Confirm password"
                                                onChange={(e) => this.handleInputPasswordConfirm(e.target.value)}
                                                value={this.state.newPasswordConfirm}
                                                required={true}
                                            />
                                            <label htmlFor="password_confirm">{t('change_password_confirm')}</label>
                                        </div>
                                    </div>
                                </div>
                                <div className="modal-footer">
                                    <button type="submit" className="btn btn-primary" disabled={!this.state.canChangePassword}>
                                        {t('set_password_submit')}
                                    </button>
                                </div>
                            </form>
                        </div>
                    </div>
                </div>
            </div>
        );
    };
}
