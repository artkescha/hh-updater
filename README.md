# hh-updater
Автоматическое обновление даты публикации резюме и добавление суффикса к описанию должностных обязанностей на сервисе hh.ru (https://dev.hh.ru)

Для использование Вам необходимо зарегистрировать приложение на hh.ru

Все настройки в файле config.yaml:

````
client_id: <ClientID>
client_secret: <ClientSecret>
public_url: http://127.0.0.1:8090
redirect_url: http://127.0.0.1:8090/callback
state_string: pe089448bde16f09ce0ae0c3eb30862a
update_interval: 30m
dump_interval: 1h
listen_address: 127.0.0.1:8090
log_level: debug
database_path: ./database_.db
cookie_name: hhupd
cookie_encryption_key: pe69fad213bb6eaf0b54f873bd199ea3
#optional field
experience_description_suffix: .
````
