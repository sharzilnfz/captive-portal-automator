/******/ (function(modules) { // webpackBootstrap
/******/ 	// The module cache
/******/ 	var installedModules = {};
/******/
/******/ 	// The require function
/******/ 	function __webpack_require__(moduleId) {
/******/
/******/ 		// Check if module is in cache
/******/ 		if(installedModules[moduleId]) {
/******/ 			return installedModules[moduleId].exports;
/******/ 		}
/******/ 		// Create a new module (and put it into the cache)
/******/ 		var module = installedModules[moduleId] = {
/******/ 			i: moduleId,
/******/ 			l: false,
/******/ 			exports: {}
/******/ 		};
/******/
/******/ 		// Execute the module function
/******/ 		modules[moduleId].call(module.exports, module, module.exports, __webpack_require__);
/******/
/******/ 		// Flag the module as loaded
/******/ 		module.l = true;
/******/
/******/ 		// Return the exports of the module
/******/ 		return module.exports;
/******/ 	}
/******/
/******/
/******/ 	// expose the modules object (__webpack_modules__)
/******/ 	__webpack_require__.m = modules;
/******/
/******/ 	// expose the module cache
/******/ 	__webpack_require__.c = installedModules;
/******/
/******/ 	// define getter function for harmony exports
/******/ 	__webpack_require__.d = function(exports, name, getter) {
/******/ 		if(!__webpack_require__.o(exports, name)) {
/******/ 			Object.defineProperty(exports, name, { enumerable: true, get: getter });
/******/ 		}
/******/ 	};
/******/
/******/ 	// define __esModule on exports
/******/ 	__webpack_require__.r = function(exports) {
/******/ 		if(typeof Symbol !== 'undefined' && Symbol.toStringTag) {
/******/ 			Object.defineProperty(exports, Symbol.toStringTag, { value: 'Module' });
/******/ 		}
/******/ 		Object.defineProperty(exports, '__esModule', { value: true });
/******/ 	};
/******/
/******/ 	// create a fake namespace object
/******/ 	// mode & 1: value is a module id, require it
/******/ 	// mode & 2: merge all properties of value into the ns
/******/ 	// mode & 4: return value when already ns object
/******/ 	// mode & 8|1: behave like require
/******/ 	__webpack_require__.t = function(value, mode) {
/******/ 		if(mode & 1) value = __webpack_require__(value);
/******/ 		if(mode & 8) return value;
/******/ 		if((mode & 4) && typeof value === 'object' && value && value.__esModule) return value;
/******/ 		var ns = Object.create(null);
/******/ 		__webpack_require__.r(ns);
/******/ 		Object.defineProperty(ns, 'default', { enumerable: true, value: value });
/******/ 		if(mode & 2 && typeof value != 'string') for(var key in value) __webpack_require__.d(ns, key, function(key) { return value[key]; }.bind(null, key));
/******/ 		return ns;
/******/ 	};
/******/
/******/ 	// getDefaultExport function for compatibility with non-harmony modules
/******/ 	__webpack_require__.n = function(module) {
/******/ 		var getter = module && module.__esModule ?
/******/ 			function getDefault() { return module['default']; } :
/******/ 			function getModuleExports() { return module; };
/******/ 		__webpack_require__.d(getter, 'a', getter);
/******/ 		return getter;
/******/ 	};
/******/
/******/ 	// Object.prototype.hasOwnProperty.call
/******/ 	__webpack_require__.o = function(object, property) { return Object.prototype.hasOwnProperty.call(object, property); };
/******/
/******/ 	// __webpack_public_path__
/******/ 	__webpack_require__.p = "";
/******/
/******/
/******/ 	// Load entry module and return exports
/******/ 	return __webpack_require__(__webpack_require__.s = 202);
/******/ })
/************************************************************************/
/******/ ({

/***/ 202:
/***/ (function(module, exports) {

function _typeof(o) { "@babel/helpers - typeof"; return _typeof = "function" == typeof Symbol && "symbol" == typeof Symbol.iterator ? function (o) { return typeof o; } : function (o) { return o && "function" == typeof Symbol && o.constructor === Symbol && o !== Symbol.prototype ? "symbol" : typeof o; }, _typeof(o); }
function _slicedToArray(r, e) { return _arrayWithHoles(r) || _iterableToArrayLimit(r, e) || _unsupportedIterableToArray(r, e) || _nonIterableRest(); }
function _nonIterableRest() { throw new TypeError("Invalid attempt to destructure non-iterable instance.\nIn order to be iterable, non-array objects must have a [Symbol.iterator]() method."); }
function _unsupportedIterableToArray(r, a) { if (r) { if ("string" == typeof r) return _arrayLikeToArray(r, a); var t = {}.toString.call(r).slice(8, -1); return "Object" === t && r.constructor && (t = r.constructor.name), "Map" === t || "Set" === t ? Array.from(r) : "Arguments" === t || /^(?:Ui|I)nt(?:8|16|32)(?:Clamped)?Array$/.test(t) ? _arrayLikeToArray(r, a) : void 0; } }
function _arrayLikeToArray(r, a) { (null == a || a > r.length) && (a = r.length); for (var e = 0, n = Array(a); e < a; e++) n[e] = r[e]; return n; }
function _iterableToArrayLimit(r, l) { var t = null == r ? null : "undefined" != typeof Symbol && r[Symbol.iterator] || r["@@iterator"]; if (null != t) { var e, n, i, u, a = [], f = !0, o = !1; try { if (i = (t = t.call(r)).next, 0 === l) { if (Object(t) !== t) return; f = !1; } else for (; !(f = (e = i.call(t)).done) && (a.push(e.value), a.length !== l); f = !0); } catch (r) { o = !0, n = r; } finally { try { if (!f && null != t["return"] && (u = t["return"](), Object(u) !== u)) return; } finally { if (o) throw n; } } return a; } }
function _arrayWithHoles(r) { if (Array.isArray(r)) return r; }
function _defineProperty(e, r, t) { return (r = _toPropertyKey(r)) in e ? Object.defineProperty(e, r, { value: t, enumerable: !0, configurable: !0, writable: !0 }) : e[r] = t, e; }
function _toPropertyKey(t) { var i = _toPrimitive(t, "string"); return "symbol" == _typeof(i) ? i : i + ""; }
function _toPrimitive(t, r) { if ("object" != _typeof(t) || !t) return t; var e = t[Symbol.toPrimitive]; if (void 0 !== e) { var i = e.call(t, r || "default"); if ("object" != _typeof(i)) return i; throw new TypeError("@@toPrimitive must return a primitive value."); } return ("string" === r ? String : Number)(t); }
(function func(global, $, _window$location$sear) {
  var accountName;
  var userValue = '';
  var locationUrl = window.location.href;
  var lastMillSec = null;
  var countdown = 0;
  var buttonLabel = '';
  var buttonColor = '#FFFFFF';
  var onProcess = false;
  var g_remainTime;
  var g_remianFlow;
  var g_isEscape = false;
  var g_validPeriod;
  var g_delayed = 1;
  var g_syncPortalFlag = 1;
  var isHttpHttpsAuth = (_window$location$sear = window.location.search) === null || _window$location$sear === void 0 ? void 0 : _window$location$sear.includes('https=true');
  window.sessionStorage4Portal.setItem('isHttpHttpsAuth', isHttpHttpsAuth);
  var STATUS_CODES = {
    USER_NOT_EXIST: '10501',
    // 认证失败：用户不存在
    USER_OR_PASS_ERROR: '10503',
    // 认证失败：用户名或密码错误
    USER_IS_LOCKED: '10505',
    // 认证失败：用户被锁定
    CHANGE_USERPASS: '10512',
    // 认证失败：第一次需要修改密码
    VERIFY_CODE: '4009',
    // 动态验证码
    VERIFY_CODE_NO_LOCK: '4002',
    // 动态验证码没有锁定
    USERPWD_OUT_DATE: '1113' // 密码将要过期
  };
  var HTTP_HTTPS_AUTH_RESULT = {
    SUCCESS: 'SUCCESS',
    // 成功
    ALREADY: 'ALREADY',
    // 已经在线
    FAILED_REJECT: 'FAILED_REJECT',
    // 用户信息校验失败
    FAILED_OTHER: 'FAILED_OTHER',
    // 认证失败
    NOTYET: 'NOTYET',
    // 未认证
    FAILED_TIMEOUT: 'FAILED_TIMEOUT',
    // 认证超时，认证服务器无响应
    FAILED_MTU: 'FAILED_MTU',
    // 认证失败
    ERROR_PROTOCOL: 'ERROR_PROTOCOL',
    // 认证失败
    FAILED_NOROUTE: 'FAILED_NOROUTE' // 认证失败，认证服务器不可达
  };
  var authType = window.getQueryString('authType');
  var ssid = window.getQueryString('ssid');
  var esn = window.getQueryString('esn');
  var apmac = window.getQueryString('apmac');
  var armac = window.getQueryString('armac');
  var userIp = window.getQueryString('uaddress');
  var userMac = window.getQueryString('umac');
  var successUrl = decodeURIComponent(window.getQueryString('successUrl'));
  var lan = window.getQueryString('lang');
  var pushPageId = window.getQueryString('pushPageId');
  var wxtoken = window.getQueryString('urlkey');
  var info = window.getQueryString('info');
  var SAMLResponse = window.getQueryString('SAMLResponse');
  var EBAResponse = window.getQueryString('EBAResponse');
  var httpHttpsAuthResult = (window.getQueryString('result') || '').toUpperCase();
  function syncPortalAuthResultWhenHttpHttpsAuth() {
    var isSuccess = httpHttpsAuthResult === HTTP_HTTPS_AUTH_RESULT.SUCCESS;
    var param = {
      authResult: String(isSuccess),
      successUrl: successUrl
    };
    if (!isSuccess) {
      param.responseCode = httpHttpsAuthResult;
    }
    // 由UI通知后台认证结果
    $.post('/portalauth/syncPortalResult', param, function (respData) {
      var data = JSON.parse(respData || '{}');
      if (isSuccess) {
        // 认证成功：页面跳转
        g_syncPortalFlag = 1;
        g_delayed = 1;
        document.cookie = "countdown=".concat(escape(0), "; path=/");
        g_validPeriod = window.timestampToTimeString((data.data || []).validPeriod);
        successUrl = (data.data || []).successUrl;
        if ((data.data || []).userName) {
          accountName = data.data.userName;
        }
        if (window.getUrlJingHaoNum() == 2 || window.getQueryString('authType') == 2) {
          setTimeout(loginSuccess, 2000);
        } else {
          loginSuccess();
        }
      } else {
        var HTTP_HTTPS_AUTH_ERROR_MSG = _defineProperty(_defineProperty(_defineProperty(_defineProperty(_defineProperty(_defineProperty(_defineProperty(_defineProperty({}, HTTP_HTTPS_AUTH_RESULT.ALREADY, window.campus.i18n.translate('wispr_response_code_ALREADY')), HTTP_HTTPS_AUTH_RESULT.FAILED_REJECT, window.campus.i18n.translate('wispr_response_code_FAILED_REJECT')), HTTP_HTTPS_AUTH_RESULT.FAILED_OTHER, window.campus.i18n.translate('wispr_response_code_FAILED_OTHER')), HTTP_HTTPS_AUTH_RESULT.NOTYET, window.campus.i18n.translate('wispr_response_code_NOTYET')), HTTP_HTTPS_AUTH_RESULT.FAILED_TIMEOUT, window.campus.i18n.translate('wispr_response_code_FAILED_TIMEOUT')), HTTP_HTTPS_AUTH_RESULT.FAILED_MTU, window.campus.i18n.translate('wispr_response_code_FAILED_MTU')), HTTP_HTTPS_AUTH_RESULT.ERROR_PROTOCOL, window.campus.i18n.translate('wispr_response_code_ERROR_PROTOCOL')), HTTP_HTTPS_AUTH_RESULT.FAILED_NOROUTE, window.campus.i18n.translate('wispr_response_code_FAILED_NOROUTE'));
        var errMsg = HTTP_HTTPS_AUTH_ERROR_MSG[httpHttpsAuthResult] || window.campus.i18n.translate('auth_failed');
        window.showdiv(errMsg, false, function () {
          var delParams = ['https', 'result'];
          window.location.search = window.delUrlParam(window.location.search, delParams);
        });
      }
    });
  }

  // 在页面加载完成后执行
  $(document).ready(function () {
    // 如果是已经点击了同意用户须知
    var acceptReadMeFlag = window.sessionStorage4Portal.getItem('acceptReadMe');
    if (acceptReadMeFlag === 'true' && $('#agreeCheck').length && $('#agreeCheck').css('display') !== 'none') {
      $('#agreeCheck').click();
      window.sessionStorage4Portal.removeItem('acceptReadMe');
    }
    if (isHttpHttpsAuth) {
      syncPortalAuthResultWhenHttpHttpsAuth();
    }
    setInformation();
  });
  if (lan && lan !== '') {
    $('html').addClass("language_".concat(lan));
  }
  document.onkeydown = subOnkeydown;
  var countdownCookie = window.getTopWindowCookie('countdown');
  if (countdownCookie.length !== 0) {
    countdown = parseInt(countdownCookie, 10);
  }
  var lastMillSecCookie = window.getTopWindowCookie('lastMillSec');
  if (lastMillSecCookie.length !== 0) {
    lastMillSec = parseInt(lastMillSecCookie, 10);
  }
  // 解决微信连接认证注销后立即认证的问题
  if (!window.sessionStorage4Portal.getItem('afterWXLogout') && wxtoken) {
    commonLogin('', '', '11');
  }
  window.sessionStorage4Portal.removeItem('afterWXLogout');

  // 获取二维码参数
  var qrcode = window.getQueryString('QRCODE');
  var manualAuth = window.sessionStorage4Portal.getItem('manualAuth');
  // 如果有qrcode参数则自动发起认证, 16为公共二维码类型访客, manualAuth为手动认证标志位
  if (qrcode && !manualAuth) {
    commonLogin('', '', authType);
    return;
  }

  // 加密的微信链接
  if (!!info && manualAuth !== 'true') {
    commonLogin('', '', authType);
    return;
  }
  if (SAMLResponse !== null || EBAResponse !== null) {
    commonLogin('', '', authType);
    return;
  }

  // 邮箱、手机、名称、申请原因、公司信息回填
  function setInformation() {
    if (locationUrl.includes('?')) {
      var paramObj = {};
      var paramUrl = locationUrl.split('?')[1];
      var paramList = paramUrl.split('&');
      paramList.forEach(function (item) {
        var _item$split = item.split('='),
          _item$split2 = _slicedToArray(_item$split, 2),
          paramItem = _item$split2[0],
          paramData = _item$split2[1];
        paramObj[paramItem] = paramData;
      });
      if (paramObj.email) {
        $('#email').attr('value', decodeURI(paramObj.email));
      }
      if (paramObj.telephone) {
        $('#mobileNum').attr('value', decodeURI(paramObj.telephone));
      }
      if (paramObj.name) {
        $('#name').attr('value', decodeURI(paramObj.name));
      }
      if (paramObj.applyReason) {
        $('#applyReason').attr('value', decodeURI(paramObj.applyReason));
      }
      if (paramObj.company) {
        $('#company').attr('value', decodeURI(paramObj.company));
      }
    }
  }
  function getVerificationCode() {
    var ACCUOUNT_SMS_AUTH_TYPE = 15;
    var businessTypes = {
      TWO_FACTOR_AUTH: 2,
      USERNAME_SMS_AUTH_TYPE: 3
    };
    var GET_DYNAMIC_VALID_CODE_TYPE_TELEPHONE = 1;
    var data = {
      pushPageId: pushPageId,
      esn: esn,
      apmac: apmac,
      umac: userMac,
      armac: armac,
      userName: window.getInputValue('username'),
      uaddress: userIp,
      dynamicType: GET_DYNAMIC_VALID_CODE_TYPE_TELEPHONE,
      businessType: businessTypes.USERNAME_SMS_AUTH_TYPE,
      acip: window.getQueryString('ac-ip'),
      lang: lan,
      validCode: window.getInputValue('validCode'),
      authType: ACCUOUNT_SMS_AUTH_TYPE
    };
    if ($('.passwordContainer').length > 0) {
      data.businessType = businessTypes.TWO_FACTOR_AUTH;
      delete data.pushPageId;
    }
    sendRequestDynamicCode(data);
  }
  function sendRequestDynamicCode(data) {
    var ONE_MIN = 60 * 1000;
    $.ajax({
      url: '/portalauth/getDynamicValidCode',
      type: 'POST',
      data: data,
      timeout: ONE_MIN,
      dataType: 'json',
      success: function success(respData) {
        if (respData.errorcode === '0000') {
          window.showdiv(window.campus.i18n.translate('get_dynamic_valid_code_success'));
        } else if (respData.errorcode === '4004' && $('#authloginContainer').closest('.pageeditor').attr('typenum') === '32') {
          // 将当前短信账号认证若未配置短信服务器，则切换成用户名密码认证
          change2UsernameAuth();
        } else {
          window.showdiv(window.campus.i18n.translate(respData.errorcode));
        }
      }
    });
  }

  // 微信链接认证获取手机动态验证码
  function getDynamicCode() {
    var phoneNum = window.getInputValue('mobileNum');
    if (phoneNum === '') {
      window.showdiv(window.campus.i18n.translate('inputPhoneNum'));
      return;
    }
    var vCode = window.getInputValue('validCode');
    if (vCode === '') {
      window.showdiv(window.campus.i18n.translate('enterValidCode'));
      return;
    }
    var data = {
      pushPageId: pushPageId,
      esn: esn,
      apmac: apmac,
      umac: userMac,
      armac: armac,
      userName: phoneNum,
      uaddress: userIp,
      dynamicType: 1,
      businessType: 5,
      acip: window.getQueryString('ac-ip'),
      lang: lan,
      validCode: vCode
    };
    sendRequestDynamicCode(data);
  }

  // 普通认证
  function login(type) {
    if (window.isPreviewPage()) {
      return;
    }
    window.showdiv(window.campus.i18n.translate('logining'), true);
    var usernameId = 'username';
    var mobileNumId = 'mobileNum';
    var emailId = 'email';
    var passwordId = 'password';
    var agreeCheckId = 'agreeCheck';
    if (type != undefined) {
      agreeCheckId += type;
      usernameId += type;
      mobileNumId += type;
      emailId += type;
      passwordId += type;
    }
    // 校验是否勾选了用户须知
    if (!checkUserNotice(agreeCheckId)) {
      return;
    }
    var $usernameDiv = $($('.authmodel>div')[0]);
    if ($usernameDiv.length && $usernameDiv.css('display') !== 'none') {
      if ($usernameDiv.hasClass('usernameContainer')) {
        userValue = window.getInputValue(usernameId);
      } else if ($usernameDiv.hasClass('mobileContainer')) {
        userValue = window.getInputValue(mobileNumId);
      } else if ($usernameDiv.hasClass('emailContainer')) {
        userValue = window.getInputValue(emailId);
        var mailTypeFilter = /^\w+((-\w+)|(\.\w+))*@[A-Za-z0-9]+((\.|-)[A-Za-z0-9]+)*\.[A-Za-z0-9]+[A-Za-z0-9]+$/;
        if (!mailTypeFilter.test(userValue)) {
          window.showdiv(window.campus.i18n.translate('emailNumFormatError'));
          return;
        }
      }
    } else if ($('.usernameContainer').length && $('.usernameContainer').css('display') !== 'none') {
      userValue = window.getInputValue(usernameId);
    } else if ($('.mobileContainer').length && $('.mobileContainer').css('display') !== 'none') {
      userValue = window.getInputValue(mobileNumId);
    } else if ($('.emailContainer').length && $('.emailContainer').css('display') !== 'none') {
      userValue = window.getInputValue(emailId);
      var _mailTypeFilter = /^\w+((-\w+)|(\.\w+))*@[A-Za-z0-9]+((\.|-)[A-Za-z0-9]+)*\.[A-Za-z0-9]+[A-Za-z0-9]+$/;
      if (!_mailTypeFilter.test(userValue)) {
        window.showdiv(window.campus.i18n.translate('emailNumFormatError'));
        return;
      }
    } else {
      userValue = window.getInputValue(usernameId);
    }
    var passValue = window.getInputValue(passwordId);
    if (!validUserInfo(userValue, passValue)) {
      return;
    }
    accountName = userValue;
    // 认证
    commonLogin(userValue, passValue, type);
  }
  function validUserInfo(userName, passValue) {
    if (userName.length === 0 || passValue.length === 0 && $('.passwordContainer').length) {
      window.showdiv(window.campus.i18n.translate('userPwdEmpty'));
      return false;
    }
    if (passValue.length > 128 && $('.passwordContainer').length) {
      window.showdiv(window.campus.i18n.translate('pwdLong'));
      return false;
    }
    if (userName.length > 128) {
      window.showdiv(window.campus.i18n.translate('userLong'));
      return false;
    }
    return true;
  }

  // 匿名认证
  function annoyLogin() {
    if (window.isPreviewPage()) {
      return;
    }
    if ($('#agreeCheck').length > 0 && $('#agreeCheck').parent().css('display') !== 'none' && $('#agreeCheck').attr('src').indexOf('uncheck') >= 0) {
      window.showdiv(window.campus.i18n.translate('agreereadme'));
      return;
    }
    window.showdiv(window.campus.i18n.translate('logining'), true);
    var username = '~anonymous';
    var userPass = '~anonymous';
    commonLogin(username, userPass);
  }

  // 获取密码
  function getPassCode(type) {
    if (window.isPreviewPage()) {
      return;
    }
    var telephone = window.getInputValue('mobileNum').trim();
    if (telephone.length === 0) {
      window.showdiv(window.campus.i18n.translate('enterNumber'));
      return;
    }
    var validCodeId = 'validCode';
    if (type != undefined) {
      validCodeId += type;
    }
    var validCode = window.getInputValue(validCodeId).trim();
    if ($("#".concat(validCodeId)).length > 0 && validCode.length === 0) {
      var $validcodeContainer = $('.validcodeContainer');
      if ($validcodeContainer.length && $validcodeContainer.css('display') === 'block') {
        window.showdiv(window.campus.i18n.translate('enterValidCode'));
        return;
      }
      if ($validcodeContainer.length === 0) {
        window.showdiv(window.campus.i18n.translate('enterValidCode'));
        return;
      }
    }
    var params = {
      pushPageId: pushPageId,
      telephone: telephone,
      esn: esn,
      apmac: apmac,
      armac: armac,
      authType: type == undefined ? authType : type,
      ssid: ssid,
      uaddress: userIp,
      umac: userMac,
      lang: lan,
      validCode: validCode,
      acip: window.getQueryString('ac-ip'),
      registerCode: window.getInputValue('registerCode')
    };
    getSmsPassword(params);
  }
  function getSmsPassword(params) {
    var ONE_MIN = 60 * 1000;
    $.ajax({
      url: '/portalauth/getSmsPassword',
      type: 'POST',
      data: params,
      timeout: ONE_MIN,
      dataType: 'json',
      success: function success(data) {
        if (data[0].successMsg === 'success') {
          window.showdiv(window.campus.i18n.translate('smsSended'));
          countdown = 60;
          lastMillSec = null;
          settime();
          return;
        }
        var errorcode = data[0].errmsg;
        var errorInfo = window.campus.i18n.translate('smsSendedFailure');
        if (errorcode) {
          errorInfo = window.campus.i18n.translate(errorcode);
        }
        window.showdiv(errorInfo);
        window.changeValidCode();
        $('#validCode').val('');
        $('#validCode3').val('');
      },
      error: function error() {
        window.showdiv(window.campus.i18n.translate('smsSendedFailure'));
        window.changeValidCode();
        $('#validCode').val('');
        $('#validCode3').val('');
      }
    });
  }

  // 获取邮箱验证码
  function getEmailPassCode(type) {
    if (window.isPreviewPage()) {
      return;
    }
    var email = window.getInputValue('email').trim().toLowerCase();
    var mailTypeFilter = /^\w+((-\w+)|(\.\w+))*@[A-Za-z0-9]+((\.|-)[A-Za-z0-9]+)*\.[A-Za-z0-9]+[A-Za-z0-9]+$/;
    if (!mailTypeFilter.test(email)) {
      window.showdiv(window.campus.i18n.translate('emailNumFormatError'));
      return;
    }
    var validCodeId = 'validCode';
    if (type != undefined) {
      validCodeId += type;
    }
    var validCode = window.getInputValue(validCodeId).trim();
    if ($("#".concat(validCodeId)).length > 0 && validCode.length === 0) {
      var $validcodeContainer = $('.validcodeContainer');
      if ($validcodeContainer.length && $validcodeContainer.css('display') === 'block') {
        window.showdiv(window.campus.i18n.translate('enterValidCode'));
        return;
      }
      if ($validcodeContainer.length === 0) {
        window.showdiv(window.campus.i18n.translate('enterValidCode'));
        return;
      }
    }
    var params = {
      pushPageId: pushPageId,
      esn: esn,
      acip: window.getQueryString('ac-ip'),
      apmac: apmac,
      armac: armac,
      accessMac: window.getQueryString('accessMac'),
      uaddress: userIp,
      umac: userMac,
      validCode: validCode,
      ssid: ssid,
      name: window.getInputValue('name'),
      company: window.getInputValue('company'),
      applyReason: window.getInputValue('applyReason'),
      email: email,
      telephone: window.getInputValue('mobileNum'),
      customFieldList: JSON.stringify(window.getCustomFieldList().length ? window.getCustomFieldList() : undefined),
      businessType: window.getQueryString('businessType'),
      agreed: '1',
      authType: '20',
      authUrl: locationUrl
    };
    getEmailSmsPassword(params);
  }
  function getEmailSmsPassword(params) {
    var ONE_MIN = 60 * 1000;
    $.ajax({
      url: '/portalauth/getEmailPassword',
      type: 'POST',
      data: params,
      timeout: ONE_MIN,
      dataType: 'json',
      success: function success(data) {
        if (data.success) {
          var _data$data;
          var content = window.campus.i18n.translate('emailSmsSended');
          if (data.tempPassEnable && (_data$data = data.data) !== null && _data$data !== void 0 && _data$data.sessionTimeout) {
            content = window.campus.i18n.translate('emailSmsSendedPassEnable', Number(data.data.sessionTimeout) / 60);
          }
          window.showdiv(content);
          countdown = 60;
          lastMillSec = null;
          settime();
          return;
        }
        var errorcode = data.errorcode;
        var errorInfo = window.campus.i18n.translate('emailSmsSendedFailure');
        if (errorcode) {
          errorInfo = window.campus.i18n.translate(errorcode);
        }
        window.showdiv(errorInfo);
        window.changeValidCode();
        $('#validCode').val('');
        $('#validCode3').val('');
      },
      error: function error() {
        window.showdiv(window.campus.i18n.translate('emailSmsSendedFailure'));
        window.changeValidCode();
        $('#validCode').val('');
        $('#validCode3').val('');
      }
    });
  }

  // 获取密码按钮计时
  function settime() {
    var el = document.getElementById('getpassword');
    if (!el) {
      return;
    }
    if (countdown <= 0) {
      el.removeAttribute('disabled');
      $('#getpassword').css('color', buttonColor || 'white');
      el.value = buttonLabel;
    } else {
      var et = new Date().getTime() - (lastMillSec || 0);
      if (lastMillSec !== null && et > 1000) {
        countdown -= parseInt(et / 1000, 10);
        if (countdown < 0) {
          countdown = 0;
        }
      } else {
        --countdown;
      }
      el.setAttribute('disabled', true);
      $('#getpassword').css('color', '#666');
      $('#getpassword').removeClass('item_button:hover');
      el.value = "".concat(countdown, "S");
      lastMillSec = new Date().getTime();
      document.cookie = "countdown=".concat(escape(countdown), "; path=/");
      document.cookie = "lastMillSec=".concat(escape(lastMillSec), "; path=/");
      setTimeout(function () {
        settime();
      }, 1000);
    }
  }

  // 短信认证
  function SMSAuth(type) {
    if (window.isPreviewPage()) {
      return;
    }
    window.showdiv(window.campus.i18n.translate('logining'), true);
    var validCodeId = 'validCode';
    var agreeCheckId = 'agreeCheck';
    if (type != undefined) {
      validCodeId += type;
      agreeCheckId += type;
    }
    // 校验是否勾选了用户须知
    if ($("#".concat(agreeCheckId)).length > 0 && $("#".concat(agreeCheckId)).parent().css('display') != 'none' && $("#".concat(agreeCheckId)).attr('src').indexOf('uncheck') >= 0) {
      window.showdiv(window.campus.i18n.translate('agreereadme'));
      return;
    }
    var userName = window.getInputValue('mobileNum').trim();
    // 手机正则表达式
    var phoneReg = /^[+,0-9][0-9-]{0,}$/;
    if (userName.length === 0 || !phoneReg.test(userName)) {
      window.showdiv(window.campus.i18n.translate('enterNumber'));
      return;
    }
    var passwordId = 'password';
    if (type != undefined) {
      passwordId = "password".concat(type);
    }
    var userPass = window.getInputValue(passwordId).trim();
    if (userPass.length === 0) {
      window.showdiv(window.campus.i18n.translate('input_password'));
      return;
    }
    var validCode = window.getInputValue(validCodeId).trim();
    if ($("#".concat(validCodeId)).length > 0 && validCode.length === 0) {
      var $validcodeContainer = $('.validcodeContainer');
      if ($validcodeContainer.length && $validcodeContainer.css('display') === 'block') {
        window.showdiv(window.campus.i18n.translate('enterValidCode'));
        return;
      }
      if ($validcodeContainer.length === 0) {
        window.showdiv(window.campus.i18n.translate('enterValidCode'));
        return;
      }
    }
    commonLogin(userName, userPass, type);
  }

  // 邮箱认证
  function emailSMSAuth(type) {
    if (window.isPreviewPage()) {
      return;
    }
    window.showdiv(window.campus.i18n.translate('logining'), true);
    var validCodeId = 'validCode';
    var agreeCheckId = 'agreeCheck';
    if (type != undefined) {
      validCodeId += type;
      agreeCheckId += type;
    }
    // 校验是否勾选了用户须知
    if ($("#".concat(agreeCheckId)).length > 0 && $("#".concat(agreeCheckId)).parent().css('display') !== 'none' && $("#".concat(agreeCheckId)).attr('src').indexOf('uncheck') >= 0) {
      window.showdiv(window.campus.i18n.translate('agreereadme'));
      return;
    }
    var email = window.getInputValue('email').trim().toLowerCase();
    var mailTypeFilter = /^\w+((-\w+)|(\.\w+))*@[A-Za-z0-9]+((\.|-)[A-Za-z0-9]+)*\.[A-Za-z0-9]+[A-Za-z0-9]+$/;
    if (!mailTypeFilter.test(email)) {
      window.showdiv(window.campus.i18n.translate('emailNumFormatError'));
      return;
    }
    var userPass = window.getInputValue('password').trim();
    if (userPass.length === 0) {
      window.showdiv(window.campus.i18n.translate('input_password'));
      return;
    }
    var validCode = window.getInputValue(validCodeId).trim();
    if ($("#".concat(validCodeId)).length > 0 && validCode.length === 0) {
      var $validcodeContainer = $('.validcodeContainer');
      if ($validcodeContainer.length && $validcodeContainer.css('display') === 'block') {
        window.showdiv(window.campus.i18n.translate('enterValidCode'));
        return;
      }
      if ($validcodeContainer.length === 0) {
        window.showdiv(window.campus.i18n.translate('enterValidCode'));
        return;
      }
    }
    commonLogin(email, userPass, type);
  }

  // Passcode认证
  function passcodeAuth(type) {
    if (window.isPreviewPage()) {
      return;
    }
    window.showdiv(window.campus.i18n.translate('logining'), true);
    var agreeCheckId = 'agreeCheck';
    if (type != undefined) {
      agreeCheckId += type;
    }

    // 校验是否勾选了用户须知
    if ($("#".concat(agreeCheckId)).length > 0 && $("#".concat(agreeCheckId)).parent().css('display') != 'none' && $("#".concat(agreeCheckId)).attr('src').indexOf('uncheck') >= 0) {
      window.showdiv(window.campus.i18n.translate('agreereadme'));
      return;
    }
    var passcodeValue;
    if ($('#passcodeInput').length) {
      passcodeValue = window.getInputValue('passcodeInput').trim();
    } else {
      passcodeValue = $('#passcode').val() || '';
    }
    if (passcodeValue.length === 0) {
      window.showdiv(window.campus.i18n.translate('input_passcode'));
      return;
    }
    commonLogin(passcodeValue, passcodeValue, type);
  }

  // 公共二维码认证
  function qrcodeAuth() {
    if (window.isPreviewPage()) {
      return;
    }
    window.showdiv(window.campus.i18n.translate('logining'), true);
    if (qrcode && qrcode !== '') {
      commonLogin('', '', '16');
    } else {
      window.showdiv(window.campus.i18n.translate('auth_failed'));
    }
  }

  // 微信连接认证
  function WechatLinkAuth() {
    if (window.isPreviewPage()) {
      return;
    }
    window.showdiv(window.campus.i18n.translate('logining'), true);
    var username = $('#mobileNum').val();
    if (username === '') {
      // 提示输入手机号
      window.showdiv(window.campus.i18n.translate('inputPhoneNum'));
    } else {
      commonLogin(username, '', authType);
    }
  }

  // 普通认证与匿名认证的公用认证过程
  function commonLogin(username, userPass, type) {
    var params = {
      pushPageId: pushPageId,
      userPass: userPass,
      esn: esn,
      apmac: apmac,
      armac: armac,
      authType: type || (authType === '18' ? '1' : authType),
      ssid: ssid,
      uaddress: userIp,
      umac: userMac,
      accessMac: window.getQueryString('accessMac'),
      businessType: window.getQueryString('businessType'),
      acip: window.getQueryString('ac-ip'),
      agreed: '1',
      registerCode: window.getInputValue('registerCode'),
      questions: window.sessionStorage4Portal.getItem('questionnaire')
    };
    var extraParams = setCondictionParams(username, userPass, type);
    var loginPrams = $.extend({}, params, extraParams);
    loginRequestHandler(loginPrams);
  }
  function setCondictionParams(username, userPass, type) {
    var params = {};
    var loginAuthType = type || authType;
    params.dynamicValidCode = window.getInputValue('dynamicValidCode').trim();
    if ($('.multiNetContainer').css('display') !== 'none') {
      params.netType = $('#select_8_0').val();
    }
    if ($('.rsaContainer').css('display') !== 'none') {
      params.dynamicRSAToken = window.getInputValue('rsa_input').trim();
    }
    if (SAMLResponse !== null) {
      params.SAMLResponse = SAMLResponse;
    }
    if (EBAResponse !== null) {
      params.EBAResponse = EBAResponse;
    }
    var currentAuthType = parseInt(loginAuthType, 10);
    if (currentAuthType === 3 || currentAuthType === 1 || parseInt(window.getUrlJingHaoNum, 10) === 3) {
      params.validCode = window.getInputValue('validCode').trim();
      if (type != undefined) {
        params.validCode = window.getInputValue("validCode".concat(type)).trim();
      }
    }
    if (loginAuthType == '6') {
      params.passcode = username;
    }
    if (loginAuthType == '9') {
      packageOneClickAuthData(params, username);
    } else {
      params.userName = username;
    }
    if (loginAuthType == '11') {
      params.wxtoken = wxtoken;
      params.wxauth = '3';
      params.info = info;
    }
    // 账号短信认证
    if ($('.pageeditor').length > 0 && $('#authloginContainer').closest('.pageeditor').attr('typenum') === '32' && $('#getpassword').length && $('#getpassword').css('display') !== 'none') {
      params.authType = '15';
      params.dynamicValidCode = userPass;
    }
    if (loginAuthType === '16') {
      params.QRCODE = window.getQueryString('QRCODE');
    }
    return params;
  }
  function packageOneClickAuthData(params, username) {
    params.userName = username;
    if ($('.mobileContainer').css('display') !== 'none') {
      params.telephone = window.getInputValue('mobileNum');
    }
    if ($('.emailContainer').css('display') !== 'none') {
      params.email = window.getInputValue('email');
    }
    params.guestPolicyId = window.getQueryString('guestPolicyId');
    params.vaildCode = window.getInputValue('validCode').trim();
    params.name = window.getInputValue('name');
    params.company = window.getInputValue('company');
    params.applyReason = window.getInputValue('applyReason');
    var customFieldList = window.getCustomFieldList();
    if (customFieldList.length) {
      params.customFieldList = JSON.stringify(customFieldList);
    }
  }
  function loginRequestHandler(params) {
    if (onProcess) {
      return;
    }
    onProcess = true;
    var ONE_MIN = 60 * 1000;
    $.ajax({
      url: '/portalauth/login',
      type: 'POST',
      data: params,
      timeout: ONE_MIN,
      dataType: 'json',
      success: function success(data) {
        onProcess = false;
        if (data && data.data && data.errorcode === '10814' && data.data.bindPhoneFlag === 'true') {
          $('.developer_mode').show();
        }
        // 清除调查问卷
        window.sessionStorage4Portal.removeItem('questionnaire');
        // 开启了多网络并且当前有选择值
        if ($('.multiNetContainer').length && $('.multiNetContainer').css('display') !== 'none' && $('#select_8_0').val()) {
          window.sessionStorage4Portal.setItem('currentNetId', $('#select_8_0').val());
        }
        loginResponse(data);
      },
      error: function error() {
        onProcess = false;
        window.showdiv(window.campus.i18n.translate('auth_failed'));
      }
    });
  }

  // 所有类型认证完后的处理
  function loginResponse(message) {
    // 设置流量时长等
    g_remainTime = message.data.remainTime === undefined || message.data.remainTime === '0' ? '' : window.returnFloat((message.data.remainTime || 0) / 60);
    g_remianFlow = message.data.remianFlow === undefined || message.data.remianFlow === '0' ? '' : window.returnFloat((message.data.remianFlow || 0) / 1048576);
    g_isEscape = message.isEscape || '';

    // 认证失败
    if (!message) {
      window.showdiv(window.campus.i18n.translate('auth_failed'));
      authFailAndUpdateValidCode();
      return;
    }
    // 认证成功
    var token = message.token || '';
    var psessionid = message.psessionid || '';
    window.addCsrfTokenToCookie(token);
    document.cookie = "PSESSIONID=".concat(escape(psessionid), "; path=/");

    // 认证成功
    if (message.success) {
      loginSuccessCallback(message);
    } else if (message.errorcode) {
      // 认证失败,如果是密码修改页面，则直接跳回认证页面
      loginFailCallback(message);
    } else {
      window.showdiv(window.campus.i18n.translate('auth_failed'));
      clearPassword();
    }
    // 认证失败更新验证码
    if (!message.success) {
      authFailAndUpdateValidCode();
    }
  }
  var generateHideElement = function generateHideElement(name, value) {
    var tempInput = document.createElement('input');
    tempInput.type = 'hidden';
    tempInput.name = name;
    tempInput.value = value;
    return tempInput;
  };
  var doHttpHttpsAuth = function doHttpHttpsAuth(httpParam) {
    var _httpParam$urlProtoco = httpParam.urlProtocol,
      urlProtocol = _httpParam$urlProtoco === void 0 ? '' : _httpParam$urlProtoco,
      ip = httpParam.ip,
      port = httpParam.port,
      loginUrl = httpParam.loginUrl,
      logOutUrl = httpParam.logOutUrl;
    var password = window.getInputValue('password').trim();
    var userName = window.getInputValue('username').trim();
    if (authType === '2') {
      // 匿名认证
      userName = '~anonymous';
    } else if (authType === '3') {
      // 短信认证
      userName = window.getInputValue('mobileNum').trim();
    } else if (authType === '6') {
      // passcode认证
      var passcodeValue;
      if ($('#passcodeInput').length) {
        passcodeValue = window.getInputValue('passcodeInput').trim();
      } else {
        passcodeValue = $('#passcode').val() || '';
      }
      password = passcodeValue;
      userName = passcodeValue;
      if (!userName) {
        userName = window.getQueryString('userInfo');
      }
      if (!password) {
        password = $('#newPasswd').val();
      }
    }
    if (!userName) {
      userName = window.getQueryString('userInfo');
    }
    if (!password) {
      var _$;
      password = (_$ = $('#newPasswd')) === null || _$ === void 0 ? void 0 : _$.val();
    }
    var url = "".concat(urlProtocol.toLowerCase(), "://").concat(ip, ":").concat(port).concat(loginUrl);
    var logoutFullUrl = "".concat(urlProtocol.toLowerCase(), "://").concat(ip, ":").concat(port).concat(logOutUrl);
    // 记录logoutUrl
    window.sessionStorage4Portal.setItem('logoutUrl', logoutFullUrl);
    var form = document.createElement('form');
    document.body.appendChild(form);
    form.action = url;
    form.method = 'GET';
    form.target = '_top';
    form.appendChild(generateHideElement('username', userName));
    if (password) {
      form.appendChild(generateHideElement('password', password));
    }
    form.submit();
  };
  function doAfterLoginSuccess(data) {
    isHttpHttpsAuth = Boolean(data.httpParam);
    window.sessionStorage4Portal.setItem('isHttpHttpsAuth', isHttpHttpsAuth);
    if (isHttpHttpsAuth) {
      window.localStorage4Portal.setItem('remainTime', g_remainTime);
      window.localStorage4Portal.setItem('remainFlow', g_remianFlow);
      window.localStorage4Portal.setItem('isEscape', g_isEscape);

      // 后台返回的httpParam是个string
      var httpParamObj = JSON.parse(data.httpParam);
      doHttpHttpsAuth(httpParamObj);
    } else {
      syncPortalAuthResult();
    }
  }
  function loginSuccessCallback(message) {
    var data = message.data;
    if (message.errorcode && message.errorcode === STATUS_CODES.USERPWD_OUT_DATE) {
      clearPassword();
      var remainDays = parseInt(data.pwdHistoryDay, 10);
      if (Number.isNaN(remainDays)) {
        window.showdiv(window.campus.i18n.translate('userpwd_out_date'), false, function () {
          doAfterLoginSuccess(data);
        });
      } else {
        window.showdiv(window.campus.i18n.translate(message.errorcode, remainDays), false, function () {
          doAfterLoginSuccess(data);
        });
      }
    } else {
      window.setTimeout(function () {
        doAfterLoginSuccess(data);
      }, 1000);
    }
  }
  function loginFailCallback(message) {
    var data = message.data;
    if (isChangePwdpage()) {
      window.showdiv(window.campus.i18n.translate(message.errorcode), false, window.turnSuccessPage);
    } else if (message.errorcode === STATUS_CODES.USER_OR_PASS_ERROR || message.errorcode === STATUS_CODES.USER_NOT_EXIST) {
      if (data.remainTimes === undefined || data.remainTimes === null || data.remainTimes === '') {
        window.showdiv(window.campus.i18n.translate(STATUS_CODES.USER_OR_PASS_ERROR));
      } else if (data.lockTime === undefined || data.lockTime === null || data.lockTime === '') {
        var userpass_times_error = window.campus.i18n.translate('user_or_pass_error_partOne', data.remainTimes);
        window.showdiv(userpass_times_error);
      } else if (data.remainTimes === '-1') {
        var lockingInfo = window.campus.i18n.translate(STATUS_CODES.USER_IS_LOCKED, data.lockTime);
        window.showdiv(lockingInfo);
      } else {
        var _userpass_times_error = window.campus.i18n.translate('user_or_pass_error_partTwo', data.remainTimes, data.lockTime);
        window.showdiv(_userpass_times_error);
      }
    } else if (message.errorcode === STATUS_CODES.USER_IS_LOCKED) {
      var remainLockTime = data.remainLockTime ? data.remainLockTime : 0;
      window.showdiv(window.campus.i18n.translate(message.errorcode, remainLockTime));
    } else if (message.errorcode === STATUS_CODES.CHANGE_USERPASS) {
      var csrfToken = message.token ? message.token : '';
      var psessionId = message.psessionid ? message.psessionid : '';
      window.addCsrfTokenToCookie(csrfToken);
      document.cookie = "PSESSIONID=".concat(escape(psessionId), "; path=/");
      window.showdiv(window.campus.i18n.translate(message.errorcode), false, turnPasspage);
    } else if (message.errorcode === STATUS_CODES.VERIFY_CODE) {
      window.showdiv(window.campus.i18n.translate(STATUS_CODES.VERIFY_CODE_NO_LOCK));
    } else {
      window.showdiv(window.campus.i18n.translate(message.errorcode));
    }
    clearPassword();
  }

  // 判断当前是否是修改密码页面
  function isChangePwdpage() {
    var changePageName = window.getPortalPage('changePassword');
    return window.location.href.indexOf(changePageName) !== -1;
  }

  // 修改密码
  function turnPasspage() {
    var changePwdPageUrl = window.getPortalPage('changePassword');
    var userName = userValue || window.getQueryString('userInfo');
    // 多种认证
    if ($('.allcheckContainer').length) {
      userName = $('#username1').val().trim();
    }
    var params = {
      userInfo: encodeURIComponent(userName),
      chanFir: 'y',
      changeUserPassFirst: 'true'
    };
    var urlParams = window.location.search;
    urlParams = window.updateUrlParams(urlParams, params);
    window.location.href = changePwdPageUrl + urlParams + window.location.hash;
  }
  function continueSyncPortalAuthResult() {
    // 轮询九次后报错
    if (g_syncPortalFlag >= 9) {
      g_delayed = 1;
      g_syncPortalFlag = 1;
      // 认证失败清空密码、更新验证码
      clearPassword();
      authFailAndUpdateValidCode();
      if (isChangePwdpage()) {
        window.showdiv(window.campus.i18n.translate('auth_failed'), false, window.turnSuccessPage);
      } else {
        window.showdiv(window.campus.i18n.translate('auth_failed'));
      }
      return;
    }
    // 轮询Portal认证结果
    if (g_syncPortalFlag > 3) {
      g_delayed++;
    }
    window.setTimeout(syncPortalAuthResult, g_delayed * 1000);
    g_syncPortalFlag++;
  }
  function syncPortalAuthResult() {
    var param = {
      successUrl: successUrl
    };
    $.post('/portalauth/syncPortalResult', param, function (respData) {
      var data = JSON.parse(respData || '{}');
      if (data.message === 'true') {
        // 认证成功：页面跳转
        g_syncPortalFlag = 1;
        g_delayed = 1;
        document.cookie = "countdown=".concat(escape(0), "; path=/");
        g_validPeriod = window.timestampToTimeString((data.data || []).validPeriod);
        successUrl = (data.data || []).successUrl;
        if ((data.data || []).userName) {
          accountName = data.data.userName;
        }
        if (window.getUrlJingHaoNum() == 2 || window.getQueryString('authType') == 2) {
          setTimeout(loginSuccess, 2000);
        } else {
          loginSuccess();
        }
      } else if (data.errorcode) {
        clearPassword();
        // 授权失败：提示错误
        if (data.errorcode == 9 || data.errorcode == 10) {
          window.showdiv(window.campus.i18n.translate(data.errorcode), false, window.turnToRightPage, data.redirectUrl);
        } else if (isChangePwdpage()) {
          window.showdiv(window.campus.i18n.translate('auth_failed') + window.campus.i18n.translate('portal_auth_reason_code') + window.campus.i18n.translate(data.errorcode), false, window.turnSuccessPage);
        } else {
          window.showdiv(window.campus.i18n.translate('auth_failed') + window.campus.i18n.translate('portal_auth_reason_code') + window.campus.i18n.translate(data.errorcode));
        }
        g_delayed = 1;
        g_syncPortalFlag = 1;
        // 认证失败更新验证码
        authFailAndUpdateValidCode();
      } else {
        continueSyncPortalAuthResult();
      }
    }).fail(function () {
      continueSyncPortalAuthResult();
    });
  }
  function change2UsernameAuth() {
    $('#getpassword').hide();
    $('#password').parent().parent().addClass('passwordContainer');
    $('#password').parent().append($('<label>'));
    $('#password').width('');
    $('#password').attr('type', 'password');
  }

  // 认证成功后页面跳转
  function loginSuccess() {
    // 推送规则页面跳转
    if (successUrl === null || successUrl === 'null' || successUrl === undefined || successUrl === '') {
      // 跳到认证成功页面
      turnAuthSuccessPage();
    } else {
      window.location.href = successUrl;
    }
  }

  // 跳转到认证成功页面
  function turnAuthSuccessPage() {
    // 多种认证
    var userName = accountName;
    if ($('.allcheckContainer').length && $('#username1').length) {
      userName = $('#username1').val();
    }
    if (window.location.href.indexOf('changePassword.html') > 0) {
      userName = window.getQueryString('userInfo');
    }
    var urlParams = {
      chanFir: 'n',
      userInfo: encodeURIComponent(userName) || encodeURIComponent(accountName),
      remainTime: g_remainTime,
      remainFlow: g_remianFlow,
      validPeriod: g_validPeriod,
      isEscape: g_isEscape
    };
    // 多网络场景支持直接推送认证成功页面，参数无法通过url传递
    if (!isHttpHttpsAuth) {
      window.localStorage4Portal.setItem('userInfo', userName);
      window.localStorage4Portal.setItem('remainTime', g_remainTime);
      window.localStorage4Portal.setItem('remainFlow', g_remianFlow);
      window.localStorage4Portal.setItem('validPeriod', g_validPeriod);
      window.localStorage4Portal.setItem('isEscape', g_isEscape);
    }
    var urlPar = window.location.search;
    var url = window.getPortalPage('authSuccess') + window.updateUrlParams(urlPar, urlParams) + window.location.hash;
    window.location.href = url;
  }

  // 点击回车自动执行认证按钮
  function subOnkeydown() {
    var _window = window,
      event = _window.event;
    if (event.keyCode == 13) {
      if ($('#_msgContainer').attr('showflag') === 'true') {
        $('#_msgContainer span.btn-wrapper').click();
        $('input').blur();
      } else if ($('.pageeditor[typenum="25"]').length) {
        // 如果是多种认证组件
        var type = window.getCurrentAuthType();
        multiLogin(type);
      } else {
        if (authType === '1') {
          login();
        } else if (authType === '2') {
          annoyLogin();
        } else if (authType === '3') {
          SMSAuth();
        } else if (authType === '6') {
          passcodeAuth();
        } else if (authType === '16') {
          qrcodeAuth();
        }
        $('input').blur();
      }
    }
    if (event.keyCode == 27 && $('#_msgContainer').attr('showflag') === 'true') {
      $('#_msgContainer .title img').click();
      $('input').blur();
    }
    if (event.keyCode == 9 || event.keyCode == 32) {
      if ($('#_msgContainer').attr('showflag') === 'true') {
        return false;
      }
    }
    return null;
  }

  // 多种认证登录
  function multiLogin(curAuthType) {
    if (curAuthType === 1) {
      login(1);
    } else if (curAuthType === 3) {
      SMSAuth(3);
    } else if (curAuthType === 6) {
      passcodeAuth(6);
    } else if (curAuthType === 5) {
      window.weixinAuth();
    } else if (curAuthType === 4) {
      window.facebookAuth();
    } else if (curAuthType === 12) {
      window.twitterAuth();
    } else if (curAuthType === 13) {
      window.googleAuth();
    } else if (curAuthType === 14 || curAuthType === 15) {
      window.sinaAndQqAuth();
    } else {
      window.login();
    }
  }
  function authFailAndUpdateValidCode() {
    var hasValidCode = ['#validCodeImg', '#validCodeImg3'].some(function (selector) {
      var validCodeDiv = $(selector).closest('div');
      return validCodeDiv.length && validCodeDiv.css('display') !== 'none';
    });
    if (hasValidCode) {
      if ($('#validCodeImg').length) {
        window.changeValidCode();
      }
      if ($('#validCodeImg3').length) {
        window.changeValidCode(3);
      }
    }
  }
  $.extend(global, {
    login: login,
    SMSAuth: SMSAuth,
    passcodeAuth: passcodeAuth,
    qrcodeAuth: qrcodeAuth,
    WechatLinkAuth: WechatLinkAuth,
    getDynamicCode: getDynamicCode,
    getPassCode: getPassCode,
    getEmailPassCode: getEmailPassCode,
    settime: settime,
    annoyLogin: annoyLogin,
    syncPortalAuthResult: syncPortalAuthResult,
    subOnkeydown: subOnkeydown,
    loginRequestHandler: loginRequestHandler,
    loginResponse: loginResponse,
    commonLogin: commonLogin,
    getVerificationCode: getVerificationCode,
    emailSMSAuth: emailSMSAuth
  });
  $(function () {
    authFailAndUpdateValidCode();
    buttonLabel = $('#getpassword').attr('value');
    buttonColor = $('#getpassword').css('color');
    if (locationUrl.indexOf('?') > 0 && window.getTopWindowCookie('countdown').length !== 0 && window.getTopWindowCookie('countdown') !== '0' && window.getTopWindowCookie('countdown') !== '60') {
      $('#getpassword').attr('value', "".concat(window.getTopWindowCookie('countdown'), "S"));
      settime();
    } else {
      $('#getpassword').attr('value', buttonLabel);
      $('#getpassword').css('color', buttonColor || 'white');
    }
  });
  function clearPassword() {
    $('#password').val('');
    $('#password1').val('');
    $('#password3').val('');
  }
  function checkUserNotice(agreeCheckId) {
    if ($("#".concat(agreeCheckId)).length > 0 && $("#".concat(agreeCheckId)).parent().css('display') != 'none' && $("#".concat(agreeCheckId)).attr('src').indexOf('uncheck') >= 0) {
      window.showdiv(window.campus.i18n.translate('agreereadme'));
      return false;
    }
    return true;
  }
})(window, $);

/***/ })

/******/ });