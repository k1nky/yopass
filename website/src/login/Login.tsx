import { useTranslation } from 'react-i18next';
import { useForm } from 'react-hook-form';
import { login, isErrorWithMessage } from '../utils/utils';
import { useState } from 'react';
import Error from '../shared/Error';
import {
//   FormGroup,
//   FormControlLabel,
  TextField,
  Typography,
  Button,
  Grid,
  Box,
  InputLabel,
} from '@mui/material';

const Login = () => {
  const { t } = useTranslation();
  const {
    // control,
    register,
    errors,
    handleSubmit,
    setError,
    clearErrors,
  } = useForm({
    defaultValues: {
        password: ''
    },
  });
  const [loading, setLoading] = useState(false);
  const [_, setResult] = useState({
    username: '',
  });

  const onKeyDown = (event: React.KeyboardEvent<HTMLInputElement>): void => {
    if (event.ctrlKey && event.key === 'Enter') {
      handleSubmit(onSubmit)();
    }
  };

  const onSubmit = async (form: any): Promise<void> => {
    // Use the manually entered password, or generate one
    setLoading(true);
    try {
      const { data, status } = await login({
        username: form.username,
        password: form.password
      });
      console.log(data, status)
      if (status !== 200) {
        setError('password', { type: 'submit', message: data.message });
      } else {
        localStorage.setItem("token", data.token);
        setResult({
            "username": "qwe123"
        });
      }
    } catch (e) {
      if (isErrorWithMessage(e)) {
        setError('password', {
          type: 'submit',
          message: e.message,
        });
      }
    }
    setLoading(false);
  };

//   if (result.username) {
//     return (
//         <Result 
//             username={result.username}/>
//     );
//   }

  return (
    <>
      <Error
        message={errors.password?.message}
        onClick={() => clearErrors('password')}
      />
      <Typography component="h1" variant="h4" align="center">
        {t('login.title')}
      </Typography>
      <form onSubmit={handleSubmit(onSubmit)}>
        <Grid container justifyContent="center" paddingTop={1}>
          <InputLabel>{t('login.inputUsernameLabel')}</InputLabel>
          <TextField
            fullWidth
            type="text"
            id="username"
            inputRef={register({ required: true })}
            name="username"
            variant="outlined"
            placeholder={t('login.inputUsernamePlaceholder')}
            inputProps={{
                autoComplete: 'off',
                spellCheck: 'false',
                'data-gramm': 'false',
            }}
          />
          <InputLabel>{t('login.inputPasswordLabel')}</InputLabel>
          <TextField
            fullWidth
            type="password"
            id="password"
            inputRef={register({ required: true })}
            name="password"
            variant="outlined"
            placeholder={t('login.inputPasswordPlaceholder')}
            onKeyDown={onKeyDown}
            inputProps={{
                autoComplete: 'off',
                spellCheck: 'false',
                'data-gramm': 'false',
            }}
          />
          <Grid container justifyContent="center">
            <Box p={2} pb={4}>
              <Button
                onClick={() => handleSubmit(onSubmit)()}
                variant="contained"
                disabled={loading}
              >
                {loading ? (
                  <span>{t('login.buttonLoginLoading')}</span>
                ) : (
                  <span>{t('login.buttonLogin')}</span>
                )}
              </Button>
            </Box>
          </Grid>
        </Grid>
      </form>
    </>
  );
};

export default Login;