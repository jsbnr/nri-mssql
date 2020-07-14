# New Relic Infrastructure Integration for Microsoft SQL Server

The New Relic Infrastructure Integration for MS SQL Server captures critical performance metrics and inventory reported by a SQL Server Instance. Data on the SQL Server Instance and Databases is collected.

Inventory and metric data is collected via SQL queries to the Instance.

See our [documentation web site](https://docs.newrelic.com/docs/integrations/host-integrations/host-integrations-list/mssql-monitoring-integration) for more details.

## Requirements

No additional requirements to monitor.

## Configuration

A user with the necessary permissions to collect all the metrics and inventory can be configured as follows

```
USE master;
CREATE LOGIN newrelic WITH PASSWORD = 'tmppassword';
CREATE USER newrelic FOR LOGIN newrelic;
GRANT CONNECT SQL TO newrelic;
GRANT VIEW SERVER STATE TO newrelic;

-- Goes through each user database and adds public permissions
DECLARE @name NVARCHAR(max)
DECLARE db_cursor CURSOR FOR
SELECT NAME
FROM master.dbo.sysdatabases
WHERE NAME NOT IN ('master','msdb','tempdb','model')
OPEN db_cursor
FETCH NEXT FROM db_cursor INTO @name WHILE @@FETCH_STATUS = 0
BEGIN
	EXECUTE('USE "' + @name + '"; CREATE USER newrelic FOR LOGIN newrelic;' );
	FETCH next FROM db_cursor INTO @name
END
CLOSE db_cursor
DEALLOCATE db_cursor
```

## Installation

- download an archive file for the `MSSQL` Integration
- extract `mssql-definition.yml` and `/bin` directory into `/var/db/newrelic-infra/newrelic-integrations`
- add execute permissions for the binary file `nri-mssql` (if required)
- extract `mssql-config.yml.sample` into `/etc/newrelic-infra/integrations.d`

## Usage

This is the description about how to run the MSSQL Integration with New Relic Infrastructure agent, so it is required to have the agent installed (see [agent installation](https://docs.newrelic.com/docs/infrastructure/new-relic-infrastructure/installation/install-infrastructure-linux)).

In order to use the MSSQL Integration it is required to configure `mssql-config.yml.sample` file. Firstly, rename the file to `mssql-config.yml`. Then, depending on your needs, specify all instances that you want to monitor. Once this is done, restart the Infrastructure agent.

You can view your data in Insights by creating your own custom NRQL queries. To do so use the **MssqlDatabaseSample**, **MssqlInstanceSample** event type.

## Custom Queries

To add custom queries, use the **-custom_metrics_query** option to provide a single query, or the **-custom_metrics_config** option to specify a YAML file with one or more queries, such as the sample `mssql-custom-query.yml.sample`

### How attributes are named

Each query that returns a table of values will be parsed row by row, adding the **MssqlCustomQuerySample** event as follows:

- The column name is the attribute name
- Each row value in that column is the attribute value
- The metric type is auto-detected whether it is a number (type GAUGE), or a string (type ATTRIBUTE)

One customizable attribute in each row can be configured by database values using the following names:

- The column `metric_name` specifies its attribute name
- The column `metric_value` specifies its attribute value
- The column `metric_type` specifies its metric type, i.e. `gauge` or `attribute`

For example, the following query makes attributes named `category_0`, `category_1`, `category_2` and so on.
```sql
SELECT CONCAT('category_', category_id) AS metric_name, name AS metric_value, category_type FROM syscategories
```

### Specifying queries in YAML

When using a YAML file containing queries, you can specify the following parameters for each query:

- `query` (required) contains the SQL query
- `database` (optional) Prepends `USE <database name>; ` to the SQL, and adds the database name as an attribute
- `prefix` (optional) prefix to prepend to the attribute name
- `metric_name` (optional) specify the name for the customizable attribute
- `metric_type` (optional) specify the metric type for the customizable attribute


## Encrypted password
To use an encrypted in the OHI's configuration file:
1. Encrypt and encode the password, result goes to stdout
  - `bin/nr-mssql -encrypt -rsa_private_key <full_path_to_rsa_private_key> -password <password>`
2. Use the result of #1 in the OHI's config file as the `password`
3. Add `rsa_private_key: <full_path_to_rsa_private_key>` to the OHI's config file
4. Check your work with 
  - `bin/nr-mssql -verbose -rsa_private_key <full_path_to_rsa_private_key> -username <username> -password <encoded_password_from_step_1`
5. Ensure the RSA private key is available to the OHI

_NOTE_: this does not provide over the wire encryption of the password to the database. This allows the OHI to use an encrypted password in the configuration file.


## Compatibility

* Supported OS: Windows version compatible with the New Relic Infrastructure Agent
* MS SQL Server versions: SQL Server 2008 R2+

Note:  It also seems to work on Linux for the containerized Linux version of MSSQL

## Integration Development usage

Assuming that you have source code you can build and run the MSSQL Integration locally.

* Go to directory of the MSSQL Integration and build it
```bash
$ make
```
* The command above will execute tests for the MSSQL Integration and build an executable file called `nri-mssql` in `bin` directory.
```bash
$ ./bin/nri-mssql
```
* If you want to know more about usage of `./nri-mssql` check
```bash
$ ./bin/nri-mssql -help
```

For managing external dependencies [govendor tool](https://github.com/kardianos/govendor) is used. It is required to lock all external dependencies to specific version (if possible) into vendor directory.
