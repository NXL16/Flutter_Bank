import 'package:http/http.dart' as http;

import 'client_factory_native.dart'
    if (dart.library.js_interop) 'client_factory_web.dart'
    as platform;

http.Client createHttpClient() => platform.createHttpClient();
