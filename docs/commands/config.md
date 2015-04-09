#### generate config

```
NAME:
   config - Generate configuration "/home/alexa/.mc/config.json" file.

USAGE:
   command config [command options] [arguments...]

DESCRIPTION:
   Configure minio client configuration data. If your config
   file does not exist (the default location is ~/.auth), it will be
   automatically created for you. Note that the configure command only writes
   values to the config file. It does not use any configuration values from
   the environment variables.

OPTIONS:
   --accesskeyid, -a	AWS access key ID
   --secretkey, -s 	AWS secret access key
   --alias 		Add aliases into config
   --completion	        Generate bash completion "/home/alexa/.mc/mc.bash_completion" file.

```