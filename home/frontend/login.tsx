import './login.scss';

import '@fontsource/roboto/300.css';
import '@fontsource/roboto/400.css';
import '@fontsource/roboto/500.css';
import '@fontsource/roboto/700.css';

import * as ReactDOM from 'react-dom';

import LoginApp, {LoginAppBootstrap} from './login/login';

(() => {
    const url = new URL(location.href);
    const features = (url.searchParams.get('use') || '').split(',');
    const useOldUi = !!features.find((e) => e == 'bs');

    ReactDOM.render(useOldUi ? <LoginAppBootstrap /> : <LoginApp />, document.getElementById('app'));
})();
